package auth

import "context"

type UserStore interface {
	CreateUser(ctx context.Context, email string, hash string) (string, error)
}

type Service struct {
	Store UserStore
}
