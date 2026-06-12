package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func AnalysisBatchCreate(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	maxFiles := int(envInt64("MAX_BATCH_FILES", 10))
	if maxFiles < 1 {
		maxFiles = 1
	}
	if maxFiles > 50 {
		maxFiles = 50
	}
	if err := r.ParseMultipartForm(envInt64("MULTIPART_MEMORY_MB", 32) * 1024 * 1024); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid multipart batch upload")
		return
	}
	defer cleanupMultipartForm(r)

	headers := r.MultipartForm.File["files"]
	if len(headers) == 0 {
		headers = r.MultipartForm.File["file"]
	}
	if len(headers) == 0 {
		estimateWriteError(w, http.StatusBadRequest, "at least one file is required")
		return
	}
	if len(headers) > maxFiles {
		estimateWriteError(w, http.StatusBadRequest, fmt.Sprintf("maximum files per batch is %d", maxFiles))
		return
	}

	var reservedBytes int64
	for _, header := range headers {
		if header.Size <= 0 {
			estimateWriteError(w, http.StatusBadRequest, fmt.Sprintf("file %q is empty", header.Filename))
			return
		}
		reservedBytes += header.Size
	}

	rawProvider := strings.TrimSpace(r.FormValue("provider"))
	rawModel := strings.TrimSpace(r.FormValue("model"))
	requestedProvider, requestedModel := resolveRequestedProvider(rawProvider, rawModel)
	provider, err := configuredAIProvider(requestedProvider, requestedModel)
	if err != nil {
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if rawProvider != "" || rawModel != "" {
		if !requestProviderOverrideAllowed() {
			estimateWriteError(w, http.StatusForbidden, "AI provider override is disabled")
			return
		}
		if err := validateRequestedAIModel(provider.Name(), rawModel, provider.Model()); err != nil {
			estimateWriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	aiJobs := int64(0)
	if provider.Name() != "rules" {
		aiJobs = int64(len(headers))
	}
	usage := UsageDelta{
		UploadFiles:  int64(len(headers)),
		UploadBytes:  reservedBytes,
		AIJobs:       aiJobs,
		BatchJobs:    1,
		StorageBytes: reservedBytes,
	}
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

	batchID := newDatabaseID("bat")
	batchDir := filepath.Join(estimateUploadDir(), "batches", batchID)
	if err := os.MkdirAll(batchDir, 0o750); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create batch directory")
		return
	}

	paths := make([]string, 0, len(headers))
	files := make([]AnalysisBatchFile, 0, len(headers))
	var actualBytes int64
	for _, header := range headers {
		file, err := saveBatchUpload(batchID, batchDir, header)
		if err != nil {
			for _, path := range paths {
				_ = os.Remove(path)
			}
			_ = os.Remove(batchDir)
			estimateWriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		paths = append(paths, file.FilePath)
		files = append(files, file)
		actualBytes += file.FileSize
	}
	if actualBytes != reservedBytes {
		if err := reconcileReservedUploadSize(r.Context(), user.ID, &usage, reservedBytes, actualBytes); err != nil {
			for _, path := range paths {
				_ = os.Remove(path)
			}
			_ = os.Remove(batchDir)
			if !writeUsageQuotaError(w, err) {
				estimateWriteError(w, http.StatusServiceUnavailable, "usage accounting is unavailable")
			}
			return
		}
	}

	batch := AnalysisBatch{
		ID:        batchID,
		Status:    "pending",
		Provider:  provider.Name(),
		Model:     provider.Model(),
		FileCount: len(files),
		CreatedAt: time.Now().UTC(),
	}
	if err := pgCreateAnalysisBatch(r.Context(), user.ID, batch, files); err != nil {
		for _, path := range paths {
			_ = os.Remove(path)
		}
		_ = os.Remove(batchDir)
		estimateWriteError(w, http.StatusInternalServerError, "cannot create analysis batch")
		return
	}
	usageReserved = false

	w.Header().Set("Location", "/v1/analysis-batches/"+batchID)
	estimateWriteJSON(w, http.StatusAccepted, map[string]any{
		"batch": batch,
		"files": publicBatchFiles(files),
	})
}

func AnalysisBatchRouter(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	batchID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/analysis-batches/"), "/")
	if batchID == "" || strings.Contains(batchID, "/") {
		estimateWriteError(w, http.StatusNotFound, "analysis batch not found")
		return
	}
	batch, files, found, err := pgFindAnalysisBatch(r.Context(), user.ID, batchID)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load analysis batch")
		return
	}
	if !found {
		estimateWriteError(w, http.StatusNotFound, "analysis batch not found")
		return
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{
		"batch": batch,
		"files": publicBatchFiles(files),
	})
}
