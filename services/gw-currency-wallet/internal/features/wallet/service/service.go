package service

import (
	"context"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func New(repository Repository) *Service {
	return &Service{
		repo: repository,
	}
}

func (s *Service) GetBalances(ctx context.Context, userID uuid.UUID) ([]domain.Balance, error) {
	if err := ctx.Err(); err != nil {
		return []domain.Balance{}, err
	}
	return s.repo.GetBalances(ctx, userID)
}

func (s *Service) ApplyBalanceOperation(ctx context.Context, operation domain.BalanceOperation) ([]domain.Balance, error) {
	if err := ctx.Err(); err != nil {
		return []domain.Balance{}, err
	}

	if err := s.repo.ApplyBalanceOperation(ctx, operation); err != nil {
		return []domain.Balance{}, err
	}

	return s.repo.GetBalances(ctx, operation.UserID())
}
