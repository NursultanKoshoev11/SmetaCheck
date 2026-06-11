package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type RequestUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func currentRequestUser(r *http.Request) (RequestUser, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return RequestUser{}, false
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return RequestUser{}, false
	}
	tokenText := strings.TrimSpace(header[7:])
	token, err := jwt.Parse(tokenText, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return RequestUser{}, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return RequestUser{}, false
	}
	user := RequestUser{}
	if value, ok := claims["sub"].(string); ok { user.ID = value }
	if value, ok := claims["email"].(string); ok { user.Email = value }
	if value, ok := claims["name"].(string); ok { user.Name = value }
	return user, user.ID != ""
}

func requireUserForProduction(w http.ResponseWriter, r *http.Request) (RequestUser, bool) {
	user, ok := currentRequestUser(r)
	if !ok {
		estimateWriteError(w, http.StatusUnauthorized, "authentication required")
		return RequestUser{}, false
	}
	return user, true
}
