package api

import (
	"net/http"
	"time"
)

func AuthMFAEnable(w http.ResponseWriter, r *http.Request) {
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
	if err := pool.QueryRow(r.Context(), `SELECT COALESCE(mfa_pending_secret_encrypted,'') FROM users WHERE id=$1`, user.ID).Scan(&encrypted); err != nil || encrypted == "" {
		estimateWriteError(w, http.StatusBadRequest, "start MFA setup first")
		return
	}
	secret, err := decryptMFASecret(encrypted)
	if err != nil || !validTOTP(secret, code, time.Now().UTC()) {
		estimateWriteError(w, http.StatusBadRequest, "invalid MFA code")
		return
	}
	if _, err := pool.Exec(r.Context(), `UPDATE users SET mfa_enabled=true,mfa_secret_encrypted=mfa_pending_secret_encrypted,mfa_pending_secret_encrypted=NULL,mfa_enabled_at=now(),updated_at=now() WHERE id=$1`, user.ID); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot enable MFA")
		return
	}
	_, _ = pool.Exec(r.Context(), `UPDATE auth_sessions SET mfa_verified_at=now() WHERE id=$1 AND user_id=$2`, user.SessionID, user.ID)
	_ = writeAuditLog(r.Context(), r, user.ID, "auth.mfa.enabled", "user", user.ID, nil)
	estimateWriteJSON(w, http.StatusOK, map[string]any{"mfa_enabled": true, "mfa_verified": true})
}
