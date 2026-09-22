package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"wallet-app/internal/core/domain"
)

type Service struct {
	repository      UserRepository
	userIDGenerator UserIDGenerator
	passwordHasher  PasswordHasher
	jwtGenerator    JWTGenerator
}

func New(repository UserRepository, userIDGenerator UserIDGenerator, passwordHasher PasswordHasher, jwtGenerator JWTGenerator) *Service {
	return &Service{
		repository:      repository,
		userIDGenerator: userIDGenerator,
		passwordHasher:  passwordHasher,
		jwtGenerator:    jwtGenerator,
	}
}

func (s *Service) Register(ctx context.Context, username string, email string, password string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	if err := s.validatePassword(password); err != nil {
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

func (s *Service) validatePassword(password string) error {

	if len(password) > s.passwordHasher.MaxByteLength() {
		return domain.ErrInvalidPassword
	}

	trimmed := strings.TrimSpace(password)

	if trimmed == "" {
		return domain.ErrInvalidPassword
	}

	return nil
}

func (s *Service) Login(ctx context.Context, username string, password string) (domain.User, string, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, "", err
	}

	if err := s.validatePassword(password); err != nil {
		return domain.User{}, "", fmt.Errorf("validate password: %w", err)
	}

	normalizedUsername := strings.ToLower(strings.TrimSpace(username))

	user, err := s.repository.GetByUsername(ctx, normalizedUsername)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, "", domain.ErrInvalidUserCredentials
		}
		return domain.User{}, "", err
	}

	if err := s.passwordHasher.Compare(password, user.PasswordHash()); err != nil {
		if errors.Is(err, domain.ErrPasswordMismatch) {
			return domain.User{}, "", domain.ErrInvalidUserCredentials
		}
		return domain.User{}, "", err
	}

	token, err := s.jwtGenerator.Generate(user.ID().String())
	if err != nil {
		return domain.User{}, "", fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}
