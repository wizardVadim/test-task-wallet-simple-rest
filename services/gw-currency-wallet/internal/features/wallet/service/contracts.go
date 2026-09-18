package service

import (
	"context"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

type Repository interface {
	CreateNewWallet(ctx context.Context, walletID domain.WalletID) (domain.Wallet, error)
	GetWalletBalance(ctx context.Context, walletID domain.WalletID) (int64, error)
	ApplyOperation(ctx context.Context, operation domain.WalletOperation) error
}

type WalletIDGenerator func() uuid.UUID
