package repository

import (
	"context"
	"errors"
	"fmt"
	"wallet-app/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresRepository struct {
	db DBTX
}

func NewPostgresRepository(db DBTX) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateNewWallet(ctx context.Context, walletID domain.WalletID) (domain.Wallet, error) {
	if err := ctx.Err(); err != nil {
		return domain.Wallet{}, err
	}

	queryRow := `
		INSERT INTO wallets (id)
		VALUES ($1)
		RETURNING balance;
	`

	var balance int64
	if err := r.db.QueryRow(ctx, queryRow, walletID.Value()).Scan(&balance); err != nil {
		return domain.Wallet{}, fmt.Errorf("create wallet: %w", err)
	}

	wallet, err := domain.NewWallet(walletID, balance)
	if err != nil {
		return domain.Wallet{}, err
	}

	return wallet, nil
}

func (r *PostgresRepository) GetWalletBalance(ctx context.Context, walletID domain.WalletID) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	queryRow := `
		SELECT balance 
		FROM wallets
		WHERE id=$1;
	`

	var balance int64

	err := r.db.QueryRow(ctx, queryRow, walletID.Value()).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrWalletNotFound
		}

		return 0, fmt.Errorf("get wallet balance: %w", err)
	}

	return balance, nil
}

func (r *PostgresRepository) GetWalletBalanceForUpdate(ctx context.Context, walletID domain.WalletID) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	queryRow := `
		SELECT balance 
		FROM wallets
		WHERE id=$1
		FOR UPDATE;
	`

	var balance int64

	err := r.db.QueryRow(ctx, queryRow, walletID.Value()).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, domain.ErrWalletNotFound
		}

		return 0, fmt.Errorf("get wallet balance for update: %w", err)
	}

	return balance, nil
}

func (r *PostgresRepository) UpdateBalance(ctx context.Context, wallet domain.Wallet) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	queryRow := `
		UPDATE wallets
		SET balance=$1
		WHERE id=$2
	`

	tag, err := r.db.Exec(ctx, queryRow, wallet.Balance(), wallet.ID().Value())
	if err != nil {
		return fmt.Errorf("update wallet balance: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrWalletNotFound
	}

	return nil
}
