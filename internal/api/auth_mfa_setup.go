package api

import (
	"crypto/rand"
	"encoding/base32"
	"net/http"
	"net/url"
)

func AuthMFASetup(w http.ResponseWriter, r *http.Request) {
	user, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	secretBytes := make([]byte, 20)
	if _, err := rand.Read(secretBytes); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create MFA secret")
		return
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	encrypted, err := encryptMFASecret(secret)
	if err != nil {
		estimateWriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable")
		return
	}
	if _, err := pool.Exec(r.Context(), `UPDATE users SET mfa_pending_secret_encrypted=$1,updated_at=now() WHERE id=$2`, encrypted, user.ID); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot save MFA setup")
		return
	}
	account := user.Email
	if account == "" { account = user.ID }
	uri := "otpauth://totp/" + url.PathEscape("SmetaCheck:"+account) + "?secret=" + url.QueryEscape(secret) + "&issuer=SmetaCheck&algorithm=SHA1&digits=6&period=30"
	_ = writeAuditLog(r.Context(), r, user.ID, "auth.mfa.setup_started", "user", user.ID, nil)
	estimateWriteJSON(w, http.StatusOK, map[string]any{"secret": secret, "otpauth_uri": uri})
}
