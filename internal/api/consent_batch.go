package api

import "net/http"

func AnalysisBatchCreateWithConsent(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUserForProduction(w, r); !ok {
		return
	}
	if err := r.ParseMultipartForm(envInt64("MULTIPART_MEMORY_MB", 32) * 1024 * 1024); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid multipart batch upload")
		return
	}
	if _, err := requireUploadConsent(r); err != nil {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	AnalysisBatchCreate(w, r)
}
