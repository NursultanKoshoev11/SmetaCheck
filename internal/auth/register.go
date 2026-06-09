package auth

import (
	"context"
	"strings"
)

func (s Service) RegisterEmail(ctx context.Context, email, password string) (string, error) {
	if !strings.Contains(email, "@") || len(password) < 8 {
		return "", ErrInvalidCredentials
	}
	hash, err := HashPassword(password)
	if err != nil { return "", err }
	return s.Store.CreateUser(ctx, email, hash)
}
