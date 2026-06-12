package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (RequestUser, bool) {
	user, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return RequestUser{}, false
	}
	return user, true
}

func EstimateUploadPostgres(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid upload form")
		return
	}
	defer cleanupMultipartForm(r)

	file, header, err := r.FormFile("file")
	if err != nil {
		estimateWriteError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()
	if header.Size <= 0 {
		estimateWriteError(w, http.StatusBadRequest, "uploaded file is empty")
		return
	}

	usage := UsageDelta{UploadFiles: 1, UploadBytes: header.Size, StorageBytes: header.Size}
	if err := reserveAccountUsage(r.Context(), user.ID, usage); err != nil {
		if !writeUsageQuotaError(w, err) {
			estimateWriteError(w, http.StatusServiceUnavailable, "usage accounting is unavailable")
		}
		return
	}
	usageReserved := true
	defer func() {
		if usageReserved {
			rollbackUsageBestEffort(user.ID, usage)
		}
	}()

	id := newEstimateID()
	fileName := sanitizeFileName(header.Filename)
	if fileName == "" {
		fileName = "estimate-file"
	}

	uploadDir := estimateUploadDir()
	reportDir := estimateReportDir()
	if err := os.MkdirAll(uploadDir, 0o750); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create upload directory")
		return
	}
	if err := os.MkdirAll(reportDir, 0o750); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create report directory")
		return
	}

	storedPath := filepath.Join(uploadDir, id+"_"+fileName)
	storedFile, err := os.OpenFile(storedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot store uploaded file")
		return
	}
	written, copyErr := io.Copy(storedFile, file)
	closeErr := storedFile.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(storedPath)
		estimateWriteError(w, http.StatusInternalServerError, "cannot save uploaded file")
		return
	}
	if written != header.Size {
		if err := reconcileReservedUploadSize(r.Context(), user.ID, &usage, header.Size, written); err != nil {
			_ = os.Remove(storedPath)
			if !writeUsageQuotaError(w, err) {
				estimateWriteError(w, http.StatusServiceUnavailable, "usage accounting is unavailable")
			}
			return
		}
	}

	items, findings := analyzeEstimateFile(storedPath, fileName, written)
	estimate := Estimate{
		ID: id, FileName: fileName, Status: estimateStatus(findings),
		Score: scoreFromFindings(findings), FileSize: written,
		UploadedAt: time.Now().UTC(), ItemsCount: len(items),
		TotalAmount: sumItemsTotal(items), Findings: findings, Items: items,
		FilePath: storedPath, ReportPath: filepath.Join(reportDir, id+"_report.txt"),
	}
	if err := writeEstimateReport(estimate); err != nil {
		_ = os.Remove(storedPath)
		estimateWriteError(w, http.StatusInternalServerError, "cannot create report")
		return
	}
	if err := pgSaveEstimate(r.Context(), user.ID, estimate); err != nil {
		_ = os.Remove(storedPath)
		_ = os.Remove(estimate.ReportPath)
		estimateWriteError(w, http.StatusInternalServerError, "cannot save estimate to postgresql")
		return
	}
	usageReserved = false
	estimateWriteJSON(w, http.StatusCreated, estimate)
}

func reconcileReservedUploadSize(ctx context.Context, userID string, usage *UsageDelta, reserved, actual int64) error {
	if actual > reserved {
		extra := actual - reserved
		delta := UsageDelta{UploadBytes: extra, StorageBytes: extra}
		if err := reserveAccountUsage(ctx, userID, delta); err != nil {
			return err
		}
		usage.UploadBytes += extra
		usage.StorageBytes += extra
		return nil
	}
	if actual < reserved {
		reduction := reserved - actual
		delta := UsageDelta{UploadBytes: reduction, StorageBytes: reduction}
		if err := rollbackAccountUsage(ctx, userID, delta); err != nil {
			return err
		}
		usage.UploadBytes -= reduction
		usage.StorageBytes -= reduction
	}
	return nil
}

func EstimateListPostgres(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	estimates, err := pgLoadEstimates(r.Context(), user.ID)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load estimates from postgresql")
		return
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{"estimates": estimates})
}

func EstimateRouterPostgres(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		EstimateDetailRouterPostgres(w, r)
	case http.MethodDelete:
		EstimateDeletePostgres(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodDelete)
		estimateWriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func EstimateDetailRouterPostgres(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1/estimates/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	id := parts[0]
	estimate, found, err := pgFindEstimate(r.Context(), user.ID, id)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load estimate from postgresql")
		return
	}
	if !found {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	if len(parts) == 2 && parts[1] == "report" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", estimate.ID+"_report.txt"))
		analysis := generateAISummary(r.Context(), user.ID, estimate)
		if err := writeAIEnhancedTextReport(w, estimate, analysis); err != nil {
			return
		}
		return
	}
	if len(parts) == 1 {
		estimateWriteJSON(w, http.StatusOK, estimate)
		return
	}
	estimateWriteError(w, http.StatusNotFound, "endpoint not found")
}

func EstimateDeletePostgres(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/estimates/"), "/")
	if path == "" || strings.Contains(path, "/") {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	filePath, reportPath, found, err := pgDeleteEstimate(r.Context(), user.ID, path)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot delete estimate from postgresql")
		return
	}
	if !found {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	if err := removeStoredFileWithinRoot(filePath, estimateUploadDir()); err != nil {
		log.Printf("estimate file cleanup failed estimate_id=%s err=%v", path, err)
	}
	if err := removeStoredFileWithinRoot(reportPath, estimateReportDir()); err != nil {
		log.Printf("estimate report cleanup failed estimate_id=%s err=%v", path, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func EstimateAISummaryPostgres(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/ai/estimate-summary/"), "/")
	if id == "" {
		estimateWriteError(w, http.StatusBadRequest, "estimate id is required")
		return
	}
	estimate, found, err := pgFindEstimate(r.Context(), user.ID, id)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load estimate from postgresql")
		return
	}
	if !found {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	estimateWriteJSON(w, http.StatusOK, generateAISummary(r.Context(), user.ID, estimate))
}

func EstimateComparePostgres(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024 * 2); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid compare form")
		return
	}
	defer cleanupMultipartForm(r)

	basePath, baseName, baseSize, err := saveCompareFile(r, "base")
	if err != nil {
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer removeTemporaryFile(basePath)

	newPath, newName, newSize, err := saveCompareFile(r, "new")
	if err != nil {
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer removeTemporaryFile(newPath)

	baseItems, baseFindings := analyzeEstimateFile(basePath, baseName, baseSize)
	newItems, newFindings := analyzeEstimateFile(newPath, newName, newSize)
	result := compareEstimateItems(baseName, newName, baseItems, newItems)
	result.Findings = append(result.Findings, baseFindings...)
	result.Findings = append(result.Findings, newFindings...)
	if err := pgSaveCompareResult(r.Context(), user.ID, result); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot save comparison to postgresql")
		return
	}
	estimateWriteJSON(w, http.StatusOK, result)
}
