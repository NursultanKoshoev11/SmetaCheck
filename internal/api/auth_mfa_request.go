package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type mfaCodeRequest struct {
	Code string `json:"code"`
}

func readMFACode(w http.ResponseWriter, r *http.Request) (string, bool) {
	var request mfaCodeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid MFA request")
		return "", false
	}
	code := strings.TrimSpace(request.Code)
	if len(code) != 6 {
		estimateWriteError(w, http.StatusBadRequest, "MFA code must contain 6 digits")
		return "", false
	}
	if _, err := strconv.Atoi(code); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "MFA code must contain 6 digits")
		return "", false
	}
	return code, true
}
