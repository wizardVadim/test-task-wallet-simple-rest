package service

import (
	"context"
	"wallet-app/internal/core/domain"
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
	walletID, err := s.walletIDGenerator()
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
