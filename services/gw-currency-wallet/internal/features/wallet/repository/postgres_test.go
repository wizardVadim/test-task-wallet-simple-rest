package repository_test

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/wallet/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupWalletDatabase(t *testing.T) (context.Context, *repository.PostgresRepository, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	t.Cleanup(cancel)
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(admin.Close)
	schema := pgx.Identifier{"wallet_test_" + uuid.New().String()}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	})

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}

	cfg.MaxConns = 8
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)
	migration, err := os.ReadFile("../../../../migrations/000001_create_wallets.up.sql")

	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	for _, name := range []string{"000002_create_users.up.sql", "000003_multicurrency_wallets.up.sql"} {
		sql, err := os.ReadFile("../../../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}

	return ctx, repository.NewPostgresRepository(pool), pool
}

func balanceOperation(t *testing.T, id uuid.UUID, code domain.CurrencyType, kind domain.OperationType, amount int64) domain.BalanceOperation {
	t.Helper()
	currency, err := domain.NewCurrency(code)
	if err != nil {
		t.Fatal(err)
	}
	op, err := domain.NewBalanceOperation(id, currency, kind, amount)
	if err != nil {
		t.Fatal(err)
	}
	return op
}

func seedBalances(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(ctx, "INSERT INTO users(id,username,email,password_hash) VALUES ($1,$2,$3,'test-hash')", id, id.String(), id.String()+"@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO balances(user_id,currency) SELECT $1,code FROM currencies", id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestApplyBalanceOperationPostgres(t *testing.T) {
	ctx, repo, pool := setupWalletDatabase(t)
	for _, tt := range []struct {
		name                  string
		initial, amount, want int64
		kind                  domain.OperationType
		wantErr               error
	}{
		{"deposit", 0, 12345, 12345, domain.OperationTypeDeposit, nil},
		{"withdraw", 12345, 45, 12300, domain.OperationTypeWithdraw, nil},
		{"withdraw all", 12345, 12345, 0, domain.OperationTypeWithdraw, nil},
		{"insufficient", 0, 1, 0, domain.OperationTypeWithdraw, domain.ErrSmallBalance},
		{"reach maximum", math.MaxInt64 - 1, 1, math.MaxInt64, domain.OperationTypeDeposit, nil},
		{"overflow", math.MaxInt64, 1, math.MaxInt64, domain.OperationTypeDeposit, domain.ErrBalanceOverflow},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id := seedBalances(t, ctx, pool)
			other := seedBalances(t, ctx, pool)
			if _, err := pool.Exec(ctx, "UPDATE balances SET amount=$1 WHERE user_id=$2 AND currency='USD'", tt.initial, id); err != nil {
				t.Fatal(err)
			}
			err := repo.ApplyBalanceOperation(ctx, balanceOperation(t, id, domain.CurrencyTypeUSD, tt.kind, tt.amount))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error=%v; want %v", err, tt.wantErr)
			}
			for _, owner := range []uuid.UUID{id, other} {
				balances, err := repo.GetBalances(ctx, owner)
				if err != nil {
					t.Fatal(err)
				}
				if len(balances) != 3 {
					t.Fatalf("got %d balances", len(balances))
				}
				for _, b := range balances {
					want := int64(0)
					if owner == id && b.Currency().CurrencyType() == domain.CurrencyTypeUSD {
						want = tt.want
					}
					if b.UserID() != owner || b.Amount() != want {
						t.Errorf("incorrect balance: owner=%v currency=%v amount=%d want=%d", b.UserID(), b.Currency(), b.Amount(), want)
					}
				}
			}
		})
	}
	for _, kind := range []domain.OperationType{domain.OperationTypeDeposit, domain.OperationTypeWithdraw} {
		if err := repo.ApplyBalanceOperation(ctx, balanceOperation(t, uuid.New(), domain.CurrencyTypeUSD, kind, 1)); !errors.Is(err, domain.ErrBalanceNotFound) {
			t.Errorf("missing user error=%v", err)
		}
	}
	id := seedBalances(t, ctx, pool)
	if _, err := pool.Exec(ctx, "DELETE FROM balances WHERE user_id=$1 AND currency='EUR'", id); err != nil {
		t.Fatal(err)
	}
	if err := repo.ApplyBalanceOperation(ctx, balanceOperation(t, id, domain.CurrencyTypeEUR, domain.OperationTypeDeposit, 1)); !errors.Is(err, domain.ErrBalanceNotFound) {
		t.Errorf("missing currency balance error=%v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := repo.ApplyBalanceOperation(canceled, balanceOperation(t, id, domain.CurrencyTypeUSD, domain.OperationTypeDeposit, 1)); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled error=%v", err)
	}
	if err := repo.ApplyBalanceOperation(ctx, domain.BalanceOperation{}); !errors.Is(err, domain.ErrInvalidOperationType) {
		t.Errorf("zero operation error=%v", err)
	}
}

func TestConcurrentBalanceOperations(t *testing.T) {
	ctx, repo, pool := setupWalletDatabase(t)
	const n = 32
	for _, tt := range []struct {
		name          string
		initial, want int64
		kind          domain.OperationType
		successes     int
		failure       error
	}{
		{"deposits", 0, n, domain.OperationTypeDeposit, n, nil},
		{"withdrawals cannot overdraw", 10, 0, domain.OperationTypeWithdraw, 10, domain.ErrSmallBalance},
		{"deposits cannot overflow", math.MaxInt64 - 10, math.MaxInt64, domain.OperationTypeDeposit, 10, domain.ErrBalanceOverflow},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id := seedBalances(t, ctx, pool)
			if _, err := pool.Exec(ctx, "UPDATE balances SET amount=$1 WHERE user_id=$2 AND currency='USD'", tt.initial, id); err != nil {
				t.Fatal(err)
			}
			op := balanceOperation(t, id, domain.CurrencyTypeUSD, tt.kind, 1)
			start := make(chan struct{})
			results := make(chan error, n)
			for i := 0; i < n; i++ {
				go func() { <-start; results <- repo.ApplyBalanceOperation(ctx, op) }()
			}
			close(start)
			successes := 0
			for i := 0; i < n; i++ {
				err := <-results
				if err == nil {
					successes++
				} else if !errors.Is(err, tt.failure) {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if successes != tt.successes {
				t.Errorf("successes=%d; want %d", successes, tt.successes)
			}
			var amount int64
			if err := pool.QueryRow(ctx, "SELECT amount FROM balances WHERE user_id=$1 AND currency='USD'", id).Scan(&amount); err != nil {
				t.Fatal(err)
			}
			if amount != tt.want {
				t.Errorf("amount=%d; want %d", amount, tt.want)
			}
		})
	}
}
