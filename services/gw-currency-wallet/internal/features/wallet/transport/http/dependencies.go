package wallet_http

import (
	"context"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

type WalletService interface {
	CreateNewWallet(ctx context.Context) (domain.Wallet, error)
	ChangeWalletBalance(ctx context.Context, operation domain.WalletOperation) error
	GetWalletBalance(ctx context.Context, walletID domain.WalletID) (int64, error)
	GetBalances(ctx context.Context, userID uuid.UUID) ([]domain.Balance, error)
	ApplyBalanceOperation(ctx context.Context, operation domain.BalanceOperation) error
}
