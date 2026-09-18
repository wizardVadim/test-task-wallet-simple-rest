package service

import (
	"context"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(context.Context, domain.User) error
	GetByUsername(context.Context, string) (domain.User, error)
}

type UserIDGenerator func() uuid.UUID

type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(string, hash string) error
}
