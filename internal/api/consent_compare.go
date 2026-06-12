package api

import "net/http"

func EstimateCompareWithConsent(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUserForProduction(w, r); !ok {
		return
	}
	if err := r.ParseMultipartForm(envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024 * 2); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid compare form")
		return
	}
	if _, err := requireUploadConsent(r); err != nil {
		cleanupMultipartForm(r)
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	EstimateComparePostgres(w, r)
}
