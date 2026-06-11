package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type RequestUser struct {
	ID        string `json:"id"`
	SessionID string `json:"-"`
	Email     string `json:"email"`
	Name      string `json:"name"`
}

func currentRequestUser(r *http.Request) (RequestUser, bool) {
	tokenText := ""
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		tokenText = strings.TrimSpace(header[7:])
	} else if cookieToken, ok := readAccessCookie(r); ok {
		tokenText = cookieToken
	}
	if tokenText == "" {
		return RequestUser{}, false
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return RequestUser{}, false
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenText, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithIssuer(authIssuer()), jwt.WithAudience("smetacheck-api"), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return RequestUser{}, false
	}

	user := RequestUser{}
	if value, ok := claims["sub"].(string); ok { user.ID = value }
	if value, ok := claims["sid"].(string); ok { user.SessionID = value }
	if value, ok := claims["email"].(string); ok { user.Email = value }
	if value, ok := claims["name"].(string); ok { user.Name = value }
	if user.ID == "" || user.SessionID == "" {
		return RequestUser{}, false
	}

	pool, err := getDB(r.Context())
	if err != nil || pool == nil {
		return RequestUser{}, false
	}
	var active bool
	err = pool.QueryRow(r.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM auth_sessions
			WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL AND expires_at>now()
		)
	`, user.SessionID, user.ID).Scan(&active)
	if err != nil || !active {
		return RequestUser{}, false
	}
	return user, true
}

func requireUserForProduction(w http.ResponseWriter, r *http.Request) (RequestUser, bool) {
	user, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return RequestUser{}, false
	}
	return user, true
}
