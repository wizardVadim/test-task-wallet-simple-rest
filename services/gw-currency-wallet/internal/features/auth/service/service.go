package service

import (
	"context"
	"fmt"
	"strings"
	"wallet-app/internal/core/domain"
)

type Service struct {
	repository      UserRepository
	userIDGenerator UserIDGenerator
	passwordHasher  PasswordHasher
}

func New(repository UserRepository, userIDGenerator UserIDGenerator, passwordHasher PasswordHasher) *Service {
	return &Service{
		repository:      repository,
		userIDGenerator: userIDGenerator,
		passwordHasher:  passwordHasher,
	}
}

func (s *Service) Register(ctx context.Context, username string, email string, password string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	if err := validatePassword(password); err != nil {
		return domain.User{}, fmt.Errorf("validate password: %w", err)
	}

	passwordHash, err := s.passwordHasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}
	userID := s.userIDGenerator()

	user, err := domain.NewUser(userID, username, email, passwordHash)
	if err != nil {
		return domain.User{}, err
	}

	if err := s.repository.Create(ctx, user); err != nil {
		return domain.User{}, err
	}

	return user, nil
}

func validatePassword(password string) error {
	validated := strings.TrimSpace(password)

	if validated == "" {
		return domain.ErrInvalidPassword
	}

	return nil
}
