package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func EstimateUploadWithConsent(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUserForProduction(w, r)
	if !ok { return }
	maxBytes := envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid upload form")
		return
	}
	consent, err := requireUploadConsent(r)
	if err != nil {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	headers := r.MultipartForm.File["file"]
	if len(headers) != 1 {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, "one file is required")
		return
	}
	header := headers[0]
	if header.Size <= 0 || header.Size > maxBytes {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, "uploaded file size is not allowed")
		return
	}
	fileName := sanitizeFileName(header.Filename)
	extension := strings.ToLower(filepath.Ext(fileName))
	temporary, err := os.CreateTemp("", "smetacheck-scan-*"+extension)
	if err != nil {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusInternalServerError, "cannot create scan file")
		return
	}
	path := temporary.Name()
	defer os.Remove(path)
	source, err := header.Open()
	if err != nil {
		temporary.Close()
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, "cannot open uploaded file")
		return
	}
	written, copyErr := io.Copy(temporary, io.LimitReader(source, maxBytes+1))
	closeSourceErr := source.Close()
	closeTempErr := temporary.Close()
	if copyErr != nil || closeSourceErr != nil || closeTempErr != nil || written > maxBytes {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, "cannot inspect uploaded file")
		return
	}
	inspection, err := inspectUploadedFile(r.Context(), path, fileName)
	if err != nil {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	provider := strings.TrimSpace(r.FormValue("provider"))
	if provider == "" { provider = "configured" }
	recordUploadConsent(r, user.ID, "upload_request", newDatabaseID("uplreq"), provider, consent)
	_ = writeAuditLog(r.Context(), r, user.ID, "file.security_scan.passed", "upload_request", fileName, map[string]any{"mime_type": inspection.MIMEType, "sha256": inspection.SHA256, "has_macros": inspection.HasMacros})
	EstimateUploadPostgres(w, r)
}
