package api

import (
	"net/http"
	"strings"
	"time"
)

type activeSession struct {
	ID string `json:"id"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	Current bool `json:"current"`
	LastUsed time.Time `json:"last_used_at"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func AuthSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := currentRequestUser(r)
	if !ok { estimateWriteError(w, http.StatusUnauthorized, "authentication required"); return }
	pool, err := getDB(r.Context())
	if err != nil || pool == nil { estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable"); return }
	rows, err := pool.Query(r.Context(), `SELECT id,COALESCE(ip_address,''),COALESCE(user_agent,''),last_used_at,created_at,expires_at FROM auth_sessions WHERE user_id=$1 AND revoked_at IS NULL AND expires_at>now() ORDER BY last_used_at DESC`, user.ID)
	if err != nil { estimateWriteError(w, http.StatusInternalServerError, "cannot load sessions"); return }
	defer rows.Close()
	sessions := make([]activeSession, 0)
	for rows.Next() {
		var item activeSession
		if err := rows.Scan(&item.ID, &item.IPAddress, &item.UserAgent, &item.LastUsed, &item.CreatedAt, &item.ExpiresAt); err != nil { estimateWriteError(w, http.StatusInternalServerError, "cannot read sessions"); return }
		item.Current = item.ID == user.SessionID
		sessions = append(sessions, item)
	}
	estimateWriteJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func AuthSessionRevoke(w http.ResponseWriter, r *http.Request) {
	user, ok := currentRequestUser(r)
	if !ok { estimateWriteError(w, http.StatusUnauthorized, "authentication required"); return }
	sessionID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/auth/sessions/"), "/")
	if sessionID == "" || strings.Contains(sessionID, "/") { estimateWriteError(w, http.StatusBadRequest, "session id is required"); return }
	pool, err := getDB(r.Context())
	if err != nil || pool == nil { estimateWriteError(w, http.StatusServiceUnavailable, "postgresql is unavailable"); return }
	command, err := pool.Exec(r.Context(), `UPDATE auth_sessions SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, sessionID, user.ID)
	if err != nil { estimateWriteError(w, http.StatusInternalServerError, "cannot revoke session"); return }
	if command.RowsAffected() == 0 { estimateWriteError(w, http.StatusNotFound, "session not found"); return }
	if sessionID == user.SessionID { clearSessionCookies(w) }
	w.WriteHeader(http.StatusNoContent)
}

func AuthRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := currentRequestUser(r)
	if !ok { estimateWriteError(w, http.StatusUnauthorized, "authentication required"); return }
	revokeAllUserSessions(r, user.ID)
	clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}
