package api

import (
	"net/http"
	"time"
)

func AuthMFAVerifySession(w http.ResponseWriter, r *http.Request) {
	user, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	code, ok := readMFACode(w, r)
	if !ok { return }
	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable")
		return
	}
	var encrypted string
	var enabled bool
	if err := pool.QueryRow(r.Context(), `SELECT COALESCE(mfa_secret_encrypted,''),mfa_enabled FROM users WHERE id=$1`, user.ID).Scan(&encrypted, &enabled); err != nil || !enabled || encrypted == "" {
		estimateWriteError(w, http.StatusBadRequest, "MFA is not enabled")
		return
	}
	secret, err := decryptMFASecret(encrypted)
	if err != nil || !validTOTP(secret, code, time.Now().UTC()) {
		estimateWriteError(w, http.StatusUnauthorized, "invalid MFA code")
		return
	}
	if _, err := pool.Exec(r.Context(), `UPDATE auth_sessions SET mfa_verified_at=now(),last_used_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, user.SessionID, user.ID); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot verify session")
		return
	}
	_ = writeAuditLog(r.Context(), r, user.ID, "auth.mfa.session_verified", "session", user.SessionID, nil)
	estimateWriteJSON(w, http.StatusOK, map[string]any{"mfa_verified": true})
}
