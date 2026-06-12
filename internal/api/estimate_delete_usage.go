package api

import (
	"log"
	"net/http"
	"os"
	"strings"
)

func EstimateRouterWithUsage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		EstimateDetailRouterPostgres(w, r)
	case http.MethodDelete:
		EstimateDeleteWithUsage(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodDelete)
		estimateWriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func EstimateDeleteWithUsage(w http.ResponseWriter, r *http.Request) {
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

	var fileSize int64
	if filePath != "" {
		if info, statErr := os.Lstat(filePath); statErr == nil && !info.IsDir() {
			fileSize = info.Size()
		}
		if err := removeStoredFileWithinRoot(filePath, estimateUploadDir()); err != nil {
			log.Printf("estimate file cleanup failed estimate_id=%s err=%v", path, err)
			fileSize = 0
		}
	}
	if fileSize > 0 {
		releaseStorageUsageBestEffort(user.ID, fileSize)
	}
	if err := removeStoredFileWithinRoot(reportPath, estimateReportDir()); err != nil {
		log.Printf("estimate report cleanup failed estimate_id=%s err=%v", path, err)
	}
	w.WriteHeader(http.StatusNoContent)
}
