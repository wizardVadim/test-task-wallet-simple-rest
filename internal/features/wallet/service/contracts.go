package service

import (
	"context"
	"wallet-app/internal/core/domain"
)

type Repository interface {
	CreateNewWallet(ctx context.Context, walletID domain.WalletID) (domain.Wallet, error)
	GetWalletBalanceForUpdate(ctx context.Context, walletID domain.WalletID) (int64, error)
	GetWalletBalance(ctx context.Context, walletID domain.WalletID) (int64, error)
	UpdateBalance(ctx context.Context, wallet domain.Wallet) error
}

type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(repo Repository) error) error
}

type WalletIDGenerator func() (domain.WalletID, error)
