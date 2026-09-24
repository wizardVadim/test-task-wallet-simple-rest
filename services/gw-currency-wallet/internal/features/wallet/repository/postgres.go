package repository

import (
	"context"
	"errors"
	"fmt"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type PostgresRepository struct {
	db DBTX
}

func NewPostgresRepository(db DBTX) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetBalances(ctx context.Context, userID uuid.UUID) ([]domain.Balance, error) {
	if err := ctx.Err(); err != nil {
		return []domain.Balance{}, err
	}

	queryRow := `
		SELECT user_id, currency, amount
		FROM balances
		WHERE user_id=$1;
	`

	rows, err := r.db.Query(ctx, queryRow, userID)
	if err != nil {
		return []domain.Balance{}, fmt.Errorf("get balances: %w", err)
	}
	defer rows.Close()

	balances := make([]domain.Balance, 0)

	for rows.Next() {
		var balanceDAO BalanceDAO
		if err := rows.Scan(
			&balanceDAO.UserID,
			&balanceDAO.Currency,
			&balanceDAO.Amount,
		); err != nil {
			return []domain.Balance{}, fmt.Errorf("get balances: %w", err)
		}

		userID, err := uuid.Parse(balanceDAO.UserID)
		if err != nil {
			return []domain.Balance{}, fmt.Errorf("%w: %+v", errors.Join(domain.ErrInvalidUserID, err), balanceDAO.UserID)
		}
		currency, err := domain.NewCurrency(domain.CurrencyType(balanceDAO.Currency))
		if err != nil {
			return []domain.Balance{}, fmt.Errorf("%w: %+v", err, balanceDAO.Currency)
		}
		balance, err := domain.NewBalance(userID, currency, balanceDAO.Amount)
		if err != nil {
			return []domain.Balance{}, err
		}

		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return []domain.Balance{}, err
	}

	return balances, nil
}

func (r *PostgresRepository) ApplyBalanceOperation(ctx context.Context, operation domain.BalanceOperation) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var query string
	var errBalanceCheck error

	switch operation.OperationType() {
	case domain.OperationTypeDeposit:
		query = `
            UPDATE balances 
            SET amount = amount + $1 
            WHERE user_id = $2 AND currency = $3 AND (9223372036854775807 - amount >= $1)
        `
		errBalanceCheck = domain.ErrBalanceOverflow

	case domain.OperationTypeWithdraw:
		query = `
            UPDATE balances 
            SET amount = amount - $1 
            WHERE user_id = $2 AND currency = $3 AND amount >= $1
        `
		errBalanceCheck = domain.ErrSmallBalance
	default:
		return domain.ErrInvalidOperationType
	}

	tag, err := r.db.Exec(ctx, query, operation.Amount(), operation.UserID().String(), operation.Currency().CurrencyType())
	if err != nil {
		return fmt.Errorf("apply balance operation: %w", err)
	}

	if tag.RowsAffected() == 0 {
		var exists bool
		checkErr := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM balances WHERE user_id=$1 AND currency=$2)", operation.UserID().String(), operation.Currency().CurrencyType()).Scan(&exists)
		if checkErr != nil {
			return fmt.Errorf("check balance existence: %w", checkErr)
		}
		if !exists {
			return domain.ErrBalanceNotFound
		}
		return errBalanceCheck
	}

	return nil
}
