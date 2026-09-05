package service

import (
	"context"
	"wallet-app/internal/core/domain"
)

type Service struct {
	repo              Repository
	txManager         TxManager
	walletIDGenerator WalletIDGenerator
}

func New(repository Repository, manager TxManager, walletIDGenerator WalletIDGenerator) *Service {
	return &Service{
		repo:              repository,
		txManager:         manager,
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

// deprecated
// func (s *Service) ChangeWalletBalance(ctx context.Context, operation domain.WalletOperation) error {
// 	if err := ctx.Err(); err != nil {
// 		return err
// 	}

// 	err := s.txManager.WithinTransaction(ctx, func(repo Repository) error {
// 		balance, err := repo.GetWalletBalanceForUpdate(ctx, operation.WalletID())
// 		if err != nil {
// 			return err
// 		}

// 		var newBalance int64
// 		switch operation.OperationType() {
// 		case domain.OperationTypeDeposit:
// 			if operation.Amount() > math.MaxInt64-balance {
// 				return ErrBalanceOverflow
// 			}
// 			newBalance = balance + operation.Amount()
// 		case domain.OperationTypeWithdraw:
// 			if balance < operation.Amount() {
// 				return ErrSmallBalance
// 			}
// 			newBalance = balance - operation.Amount()
// 		default:
// 			return domain.ErrInvalidOperationType
// 		}

// 		wallet, err := domain.NewWallet(operation.WalletID(), newBalance)
// 		if err != nil {
// 			return err
// 		}
// 		if err := repo.UpdateBalance(ctx, wallet); err != nil {
// 			return err
// 		}

// 		return nil
// 	})

// 	return err
// }

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
