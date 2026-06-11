package api

import "net/http"

func AuthLogoutSecure(w http.ResponseWriter, r *http.Request) {
	requestUser, _ := currentRequestUser(r)
	pool, err := getDB(r.Context())
	if err == nil && pool != nil {
		if requestUser.SessionID != "" {
			_, _ = pool.Exec(r.Context(), `UPDATE auth_sessions SET revoked_at=now() WHERE id=$1 AND user_id=$2`, requestUser.SessionID, requestUser.ID)
		}
		if refreshToken, ok := readRefreshCookie(r); ok {
			_, _ = pool.Exec(r.Context(), `UPDATE auth_sessions SET revoked_at=now() WHERE refresh_token_hash=$1`, hashToken(refreshToken))
		}
	}
	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}
