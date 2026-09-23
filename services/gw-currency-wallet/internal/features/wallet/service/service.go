package service

import (
	"context"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

type Service struct {
	repo              Repository
	walletIDGenerator WalletIDGenerator
}

func New(repository Repository, walletIDGenerator WalletIDGenerator) *Service {
	return &Service{
		repo:              repository,
		walletIDGenerator: walletIDGenerator,
	}
}

func (s *Service) CreateNewWallet(ctx context.Context) (domain.Wallet, error) {
	id := s.walletIDGenerator()
	walletID, err := domain.NewWalletID(id)
	if err != nil {
		return domain.Wallet{}, err
	}
	return s.repo.CreateNewWallet(ctx, walletID)
}

func (s *Service) ChangeWalletBalance(ctx context.Context, operation domain.WalletOperation) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return s.repo.ApplyOperation(ctx, operation)
}

func (s *Service) GetWalletBalance(ctx context.Context, walletID domain.WalletID) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return s.repo.GetWalletBalance(ctx, walletID)
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
