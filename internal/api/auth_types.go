package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type authResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

func createAuthToken(user User) (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET is required")
	}
	claims := jwt.MapClaims{
		"sub": user.ID,
		"email": user.Email,
		"name": user.FullName,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
