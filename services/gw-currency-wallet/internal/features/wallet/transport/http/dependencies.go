package wallet_http

import (
	"context"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

type WalletService interface {
	GetBalances(ctx context.Context, userID uuid.UUID) ([]domain.Balance, error)
	ApplyBalanceOperation(ctx context.Context, operation domain.BalanceOperation) ([]domain.Balance, error)
}
