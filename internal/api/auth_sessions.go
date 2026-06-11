package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	accessCookieDev   = "smetacheck_access"
	refreshCookieDev  = "smetacheck_refresh"
	accessCookieProd  = "__Host-smetacheck_access"
	refreshCookieProd = "__Host-smetacheck_refresh"
)

func createBrowserSession(w http.ResponseWriter, r *http.Request, user User) error {
	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		return fmt.Errorf("postgresql is unavailable")
	}
	sessionID := newDatabaseID("ses")
	accessToken, err := createAuthTokenForSession(user, sessionID)
	if err != nil {
		return err
	}
	refreshToken, err := randomURLToken(48)
	if err != nil {
		return err
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	_, err = pool.Exec(r.Context(), `
		INSERT INTO auth_sessions (
			id, user_id, refresh_token_hash, ip_address, user_agent, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6)
	`, sessionID, user.ID, hashToken(refreshToken), requestIP(r), r.UserAgent(), expiresAt)
	if err != nil {
		return err
	}
	setSessionCookies(w, accessToken, refreshToken, expiresAt)
	return nil
}

func AuthRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, ok := readRefreshCookie(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "refresh session is missing")
		return
	}
	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable")
		return
	}

	var sessionID string
	var user User
	err = pool.QueryRow(r.Context(), `
		SELECT s.id, u.id, COALESCE(u.email,''), COALESCE(u.full_name,''),
		       COALESCE(u.avatar_url,''), u.email_verified_at, u.created_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
	`, hashToken(refreshToken)).Scan(&sessionID, &user.ID, &user.Email, &user.FullName,
		&user.AvatarURL, &user.EmailVerifiedAt, &user.CreatedAt)
	if err != nil {
		clearSessionCookies(w)
		estimateWriteError(w, http.StatusUnauthorized, "refresh session is invalid")
		return
	}

	newRefresh, err := randomURLToken(48)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot rotate session")
		return
	}
	accessToken, err := createAuthTokenForSession(user, sessionID)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create access token")
		return
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	command, err := pool.Exec(r.Context(), `
		UPDATE auth_sessions
		SET refresh_token_hash=$1, expires_at=$2, last_used_at=now(),
		    ip_address=$3, user_agent=$4
		WHERE id=$5 AND refresh_token_hash=$6 AND revoked_at IS NULL AND expires_at>now()
	`, hashToken(newRefresh), expiresAt, requestIP(r), r.UserAgent(), sessionID, hashToken(refreshToken))
	if err != nil || command.RowsAffected() != 1 {
		clearSessionCookies(w)
		estimateWriteError(w, http.StatusUnauthorized, "refresh session rotation failed")
		return
	}
	setSessionCookies(w, accessToken, newRefresh, expiresAt)
	estimateWriteJSON(w, http.StatusOK, map[string]any{"user": user})
}

func AuthLogout(w http.ResponseWriter, r *http.Request) {
	if refreshToken, ok := readRefreshCookie(r); ok {
		if pool, err := getDB(r.Context()); err == nil && pool != nil {
			_, _ = pool.Exec(r.Context(), `UPDATE auth_sessions SET revoked_at=now() WHERE refresh_token_hash=$1`, hashToken(refreshToken))
		}
	}
	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func revokeAllUserSessions(r *http.Request, userID string) {
	if pool, err := getDB(r.Context()); err == nil && pool != nil {
		_, _ = pool.Exec(r.Context(), `UPDATE auth_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	}
}

func setSessionCookies(w http.ResponseWriter, accessToken, refreshToken string, refreshExpires time.Time) {
	secure := cookieSecure()
	accessName, refreshName := sessionCookieNames(secure)
	http.SetCookie(w, &http.Cookie{Name: accessName, Value: accessToken, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: 900})
	http.SetCookie(w, &http.Cookie{Name: refreshName, Value: refreshToken, Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: refreshExpires, MaxAge: int((30 * 24 * time.Hour).Seconds())})
}

func clearSessionCookies(w http.ResponseWriter) {
	for _, secure := range []bool{false, true} {
		accessName, refreshName := sessionCookieNames(secure)
		for _, name := range []string{accessName, refreshName} {
			http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
		}
	}
}

func readAccessCookie(r *http.Request) (string, bool) {
	for _, name := range []string{accessCookieProd, accessCookieDev} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return cookie.Value, true
		}
	}
	return "", false
}

func readRefreshCookie(r *http.Request) (string, bool) {
	for _, name := range []string{refreshCookieProd, refreshCookieDev} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return cookie.Value, true
		}
	}
	return "", false
}

func sessionCookieNames(secure bool) (string, string) {
	if secure {
		return accessCookieProd, refreshCookieProd
	}
	return accessCookieDev, refreshCookieDev
}

func cookieSecure() bool {
	if value := strings.TrimSpace(os.Getenv("COOKIE_SECURE")); value != "" {
		parsed, _ := strconv.ParseBool(value)
		return parsed
	}
	return strings.EqualFold(os.Getenv("APP_ENV"), "production")
}

func randomURLToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func requestIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
