package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	accessCookieDev   = "smetacheck_access"
	refreshCookieDev  = "smetacheck_refresh"
	accessCookieProd  = "__Host-smetacheck_access"
	refreshCookieProd = "__Host-smetacheck_refresh"
)

type refreshRotationResult int

const (
	refreshRotationInvalid refreshRotationResult = iota
	refreshRotationSuccess
	refreshRotationConcurrent
	refreshRotationReused
)

type rotatedSession struct {
	SessionID string
	User      User
	Result    refreshRotationResult
}

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
	`, sessionID, user.ID, hashToken(refreshToken), normalizedRequestIP(r), normalizedUserAgent(r), expiresAt)
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
	newRefresh, err := randomURLToken(48)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot rotate session")
		return
	}
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
	rotated, err := rotateRefreshToken(
		r.Context(),
		hashToken(refreshToken),
		hashToken(newRefresh),
		expiresAt,
		normalizedRequestIP(r),
		normalizedUserAgent(r),
	)
	if err != nil {
		estimateWriteError(w, http.StatusServiceUnavailable, "refresh service is unavailable")
		return
	}

	switch rotated.Result {
	case refreshRotationConcurrent:
		w.Header().Set("Retry-After", "1")
		estimateWriteError(w, http.StatusConflict, "refresh session was already rotated")
		return
	case refreshRotationReused:
		clearSessionCookies(w)
		estimateWriteError(w, http.StatusUnauthorized, "refresh token reuse detected; session revoked")
		return
	case refreshRotationInvalid:
		clearSessionCookies(w)
		estimateWriteError(w, http.StatusUnauthorized, "refresh session is invalid")
		return
	case refreshRotationSuccess:
		// Continue below.
	default:
		clearSessionCookies(w)
		estimateWriteError(w, http.StatusUnauthorized, "refresh session is invalid")
		return
	}

	accessToken, err := createAuthTokenForSession(rotated.User, rotated.SessionID)
	if err != nil {
		revokeSessionBestEffort(rotated.SessionID, "auth.refresh_access_token_failed")
		clearSessionCookies(w)
		estimateWriteError(w, http.StatusInternalServerError, "cannot create access token")
		return
	}
	setSessionCookies(w, accessToken, newRefresh, expiresAt)
	estimateWriteJSON(w, http.StatusOK, map[string]any{"user": rotated.User})
}

func rotateRefreshToken(ctx context.Context, currentHash, newHash string, newExpiresAt time.Time, ipAddress, userAgent string) (rotatedSession, error) {
	pool, err := getDB(ctx)
	if err != nil || pool == nil {
		return rotatedSession{}, fmt.Errorf("postgresql is unavailable")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return rotatedSession{}, err
	}
	defer tx.Rollback(ctx)

	var result rotatedSession
	var oldExpiresAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT s.id, s.expires_at,
		       u.id, COALESCE(u.email,''), COALESCE(u.full_name,''),
		       COALESCE(u.avatar_url,''), u.email_verified_at, u.created_at
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		FOR UPDATE OF s
	`, currentHash).Scan(
		&result.SessionID,
		&oldExpiresAt,
		&result.User.ID,
		&result.User.Email,
		&result.User.FullName,
		&result.User.AvatarURL,
		&result.User.EmailVerifiedAt,
		&result.User.CreatedAt,
	)
	if err == nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO auth_refresh_token_history (
				token_hash,session_id,user_id,rotated_at,expires_at,ip_address,user_agent
			) VALUES ($1,$2,$3,now(),$4,$5,$6)
		`, currentHash, result.SessionID, result.User.ID, oldExpiresAt, ipAddress, userAgent); err != nil {
			return rotatedSession{}, err
		}
		command, err := tx.Exec(ctx, `
			UPDATE auth_sessions
			SET previous_refresh_token_hash=$1,
			    refresh_token_hash=$2,
			    rotated_at=now(),
			    expires_at=$3,
			    last_used_at=now(),
			    ip_address=$4,
			    user_agent=$5
			WHERE id=$6 AND refresh_token_hash=$1
			  AND revoked_at IS NULL AND expires_at>now()
		`, currentHash, newHash, newExpiresAt, ipAddress, userAgent, result.SessionID)
		if err != nil || command.RowsAffected() != 1 {
			if err == nil {
				err = fmt.Errorf("refresh session changed during rotation")
			}
			return rotatedSession{}, err
		}
		result.Result = refreshRotationSuccess
		if err := tx.Commit(ctx); err != nil {
			return rotatedSession{}, err
		}
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return rotatedSession{}, err
	}

	var historySessionID, historyUserID, historyIP, historyUserAgent string
	var rotatedAt, historyExpiresAt time.Time
	var revokedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT h.session_id,h.user_id,h.rotated_at,h.expires_at,
		       COALESCE(h.ip_address,''),COALESCE(h.user_agent,''),s.revoked_at
		FROM auth_refresh_token_history h
		JOIN auth_sessions s ON s.id=h.session_id
		WHERE h.token_hash=$1
		FOR UPDATE OF s
	`, currentHash).Scan(
		&historySessionID,
		&historyUserID,
		&rotatedAt,
		&historyExpiresAt,
		&historyIP,
		&historyUserAgent,
		&revokedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return rotatedSession{Result: refreshRotationInvalid}, nil
	}
	if err != nil {
		return rotatedSession{}, err
	}

	grace := refreshReuseGracePeriod()
	withinGrace := grace > 0 && time.Since(rotatedAt) >= 0 && time.Since(rotatedAt) <= grace
	sameClient := constantTimeStringEqual(historyIP, ipAddress) && constantTimeStringEqual(historyUserAgent, userAgent)
	if withinGrace && sameClient && revokedAt == nil && historyExpiresAt.After(time.Now().UTC()) {
		if err := tx.Commit(ctx); err != nil {
			return rotatedSession{}, err
		}
		return rotatedSession{SessionID: historySessionID, Result: refreshRotationConcurrent}, nil
	}

	if revokedAt == nil {
		if _, err := tx.Exec(ctx, `
			UPDATE auth_sessions
			SET revoked_at=now()
			WHERE id=$1 AND revoked_at IS NULL
		`, historySessionID); err != nil {
			return rotatedSession{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (
			id,user_id,action,resource_type,resource_id,ip_address,user_agent
		) VALUES ($1,$2,'auth.refresh_token_reused','auth_session',$3,$4,$5)
	`, newDatabaseID("aud"), historyUserID, historySessionID, nullableString(ipAddress), nullableString(userAgent)); err != nil {
		return rotatedSession{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return rotatedSession{}, err
	}
	return rotatedSession{SessionID: historySessionID, Result: refreshRotationReused}, nil
}

func refreshReuseGracePeriod() time.Duration {
	grace := envDuration("REFRESH_REUSE_GRACE", 5*time.Second)
	if grace < 0 {
		return 0
	}
	if grace > 30*time.Second {
		return 30 * time.Second
	}
	return grace
}

func constantTimeStringEqual(left, right string) bool {
	leftHash := sha256.Sum256([]byte(left))
	rightHash := sha256.Sum256([]byte(right))
	var difference byte
	for index := range leftHash {
		difference |= leftHash[index] ^ rightHash[index]
	}
	return difference == 0
}

func normalizedRequestIP(r *http.Request) string {
	return truncateUTF8(strings.TrimSpace(requestIP(r)), 128)
}

func normalizedUserAgent(r *http.Request) string {
	return truncateUTF8(strings.TrimSpace(r.UserAgent()), 512)
}

func revokeSessionBestEffort(sessionID, action string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := getDB(ctx)
	if err != nil || pool == nil {
		return
	}
	_, _ = pool.Exec(ctx, `UPDATE auth_sessions SET revoked_at=now() WHERE id=$1 AND revoked_at IS NULL`, sessionID)
	_, _ = pool.Exec(ctx, `
		INSERT INTO audit_logs (id,action,resource_type,resource_id)
		VALUES ($1,$2,'auth_session',$3)
	`, newDatabaseID("aud"), action, sessionID)
}

func AuthLogout(w http.ResponseWriter, r *http.Request) {
	if refreshToken, ok := readRefreshCookie(r); ok {
		if pool, err := getDB(r.Context()); err == nil && pool != nil {
			tokenHash := hashToken(refreshToken)
			_, _ = pool.Exec(r.Context(), `
				UPDATE auth_sessions
				SET revoked_at=now()
				WHERE revoked_at IS NULL AND (
					refresh_token_hash=$1 OR id IN (
						SELECT session_id FROM auth_refresh_token_history WHERE token_hash=$1
					)
				)
			`, tokenHash)
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
