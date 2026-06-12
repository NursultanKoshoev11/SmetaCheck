package api

import (
	"net/http"
	"strings"
)

func requireAdministrator(w http.ResponseWriter, r *http.Request) (RequestUser, bool) {
	user, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return RequestUser{}, false
	}
	if !strings.EqualFold(user.Role, "admin") || !user.MFAEnabled || !user.MFAVerified {
		estimateWriteError(w, http.StatusForbidden, "verified administrator access required")
		return RequestUser{}, false
	}
	return user, true
}
