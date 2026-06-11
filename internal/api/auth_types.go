package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID              string     `json:"id"`
	Email           string     `json:"email,omitempty"`
	FullName        string     `json:"full_name"`
	AvatarURL       string     `json:"avatar_url,omitempty"`
	PasswordHash    string     `json:"-"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type authResponse struct {
	Token string `json:"token,omitempty"`
	User  User   `json:"user"`
}

func createAuthToken(user User) (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is required")
	}
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub": user.ID,
		"email": user.Email,
		"name": user.FullName,
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"exp": now.Add(15 * time.Minute).Unix(),
		"iss": authIssuer(),
		"aud": "smetacheck-api",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func authIssuer() string {
	issuer := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/")
	if issuer == "" {
		return "smetacheck"
	}
	return issuer
}
