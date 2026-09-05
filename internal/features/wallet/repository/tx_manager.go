package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
	"wallet-app/internal/features/wallet/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// deprecated
type PostgresTxManager struct {
	pool *pgxpool.Pool
}

func NewPostgresTxManager(pool *pgxpool.Pool) *PostgresTxManager {
	return &PostgresTxManager{pool: pool}
}

func (m *PostgresTxManager) WithinTransaction(ctx context.Context, fn func(repo service.Repository) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := tx.Rollback(rollbackCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("within transaction rollback error: %v", err)
		}
	}()

	repo := NewPostgresRepository(tx)

	if err := fn(repo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
