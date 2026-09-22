package auth_http

import (
	"context"
	"wallet-app/internal/core/domain"
)

type AuthService interface {
	Register(ctx context.Context, username string, email string, password string) (domain.User, error)
	Login(ctx context.Context, username string, password string) (domain.User, string, error)
}

type TokenValidator interface {
	Validate(token string) (string, error)
}
