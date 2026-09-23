package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type PostgresRepository struct {
	db DBTX
}

func NewPostgresRepository(db DBTX) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (repo *PostgresRepository) Create(ctx context.Context, user domain.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	queryInsertUserRow := `
		INSERT INTO users (id, username, email, password_hash)
		VALUES ($1, $2, $3, $4);
	`

	queryInsertBalancesRow := `
		INSERT INTO balances (user_id, currency)
		VALUES
		($1, 'USD'),
		($2, 'RUB'),
		($3, 'EUR');
	`

	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			if errors.Is(err, pgx.ErrTxClosed) {
				return
			}
			slog.ErrorContext(ctx, "create user: rollback error", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, queryInsertUserRow, user.ID().String(), user.Username(), user.Email(), user.PasswordHash()); err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return domain.ErrUsernameAlreadyExists
			case "users_email_lower_idx":
				return domain.ErrEmailAlreadyExists
			}
		}

		return fmt.Errorf("create user: %w", err)
	}

	if _, err := tx.Exec(ctx, queryInsertBalancesRow, user.ID().String(), user.ID().String(), user.ID().String()); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (repo *PostgresRepository) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	if err := ctx.Err(); err != nil {
		return domain.User{}, err
	}

	queryRow := `
		SELECT id, username, email, password_hash FROM users WHERE username=$1;
	`

	var userDAO UserDAO

	err := repo.db.QueryRow(ctx, queryRow, username).Scan(&userDAO.ID, &userDAO.Username, &userDAO.Email, &userDAO.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}

		return domain.User{}, fmt.Errorf("%w: %+v", err, username)
	}

	userID, err := uuid.Parse(userDAO.ID)
	if err != nil {
		return domain.User{}, fmt.Errorf("%w: %+v", err, userDAO.ID)
	}

	user, err := domain.NewUser(userID, userDAO.Username, userDAO.Email, userDAO.PasswordHash)
	if err != nil {
		return domain.User{}, fmt.Errorf("%w: %+v", err, username)
	}

	return user, nil
}
