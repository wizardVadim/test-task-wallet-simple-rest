package repository_test

import (
	"context"
	"errors"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/wallet/repository"
	"wallet-app/internal/features/wallet/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupRepository(t *testing.T) (context.Context, *repository.PostgresRepository) {
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

	return ctx, repository.NewPostgresRepository(pool)
}

func newWalletID(t *testing.T) domain.WalletID {
	t.Helper()
	id, err := domain.NewWalletID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func createWallet(t *testing.T, ctx context.Context, repo service.Repository) domain.WalletID {
	t.Helper()
	id := newWalletID(t)
	if _, err := repo.CreateNewWallet(ctx, id); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	return id
}

func withBalance(t *testing.T, id domain.WalletID, balance int64) domain.Wallet {
	t.Helper()
	wallet, err := domain.NewWallet(id, balance)
	if err != nil {
		t.Fatal(err)
	}
	return wallet
}

func withOperation(t *testing.T, id domain.WalletID, operationType domain.OperationType, amount int64) domain.WalletOperation {
	t.Helper()
	operation, err := domain.NewWalletOperation(id, operationType, amount)
	if err != nil {
		t.Fatal(err)
	}
	return operation
}

func assertBalance(t *testing.T, ctx context.Context, repo service.Repository, id domain.WalletID, want int64) {
	t.Helper()
	got, err := repo.GetWalletBalance(ctx, id)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if got != want {
		t.Fatalf("balance = %d; want %d", got, want)
	}
}

func TestCreateNewWallet(t *testing.T) {
	ctx, repo := setupRepository(t)
	id := newWalletID(t)
	wallet, err := repo.CreateNewWallet(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if wallet.ID() != id || wallet.Balance() != 0 {
		t.Fatalf("unexpected wallet: %+v", wallet)
	}
	assertBalance(t, ctx, repo, id, 0)
}

func TestGetWalletBalance(t *testing.T) {
	ctx, repo := setupRepository(t)
	id := createWallet(t, ctx, repo)
	if err := repo.ApplyOperation(ctx, withOperation(t, id, domain.OperationTypeDeposit, 1000)); err != nil {
		t.Fatal(err)
	}
	assertBalance(t, ctx, repo, id, 1000)
}

func TestWalletNotFound(t *testing.T) {
	ctx, repo := setupRepository(t)
	id := newWalletID(t)
	tests := []struct {
		name string
		run  func() error
	}{
		{"read", func() error { _, err := repo.GetWalletBalance(ctx, id); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, domain.ErrWalletNotFound) {
				t.Fatalf("error = %v; want ErrWalletNotFound", err)
			}
		})
	}
}

func TestCreateDuplicateWallet(t *testing.T) {
	ctx, repo := setupRepository(t)
	id := createWallet(t, ctx, repo)
	if err := repo.ApplyOperation(ctx, withOperation(t, id, domain.OperationTypeDeposit, 100)); err != nil {
		t.Fatal(err)
	}
	_, err := repo.CreateNewWallet(ctx, id)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("error = %v; want unique violation", err)
	}
	assertBalance(t, ctx, repo, id, 100)
}

func TestConcurrentApplyOperations(t *testing.T) {
	ctx, repo := setupRepository(t)

	id := createWallet(t, ctx, repo)

	const operations = 40

	start := make(chan struct{})
	results := make(chan error, operations)

	var workers sync.WaitGroup

	for i := 0; i < operations; i++ {
		workers.Add(1)

		go func() {
			defer workers.Done()

			<-start

			operation, err := domain.NewWalletOperation(
				id,
				domain.OperationTypeDeposit,
				1,
			)
			if err != nil {
				results <- err
				return
			}

			results <- repo.ApplyOperation(ctx, operation)
		}()
	}

	close(start)

	workers.Wait()
	close(results)

	for err := range results {
		if err != nil {
			t.Errorf("concurrent operation: %v", err)
		}
	}

	assertBalance(t, ctx, repo, id, operations)
}

func TestApplyOperationCancelledContext(t *testing.T) {
	ctx, repo := setupRepository(t)

	id := createWallet(t, ctx, repo)

	operation, err := domain.NewWalletOperation(
		id,
		domain.OperationTypeDeposit,
		100,
	)
	if err != nil {
		t.Fatal(err)
	}

	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = repo.ApplyOperation(cancelledCtx, operation)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v; want context.Canceled", err)
	}

	assertBalance(t, ctx, repo, id, 0)
}

func TestApplyOperation(t *testing.T) {
	ctx, repo := setupRepository(t)

	tests := []struct {
		name          string
		initial       int64
		operationType domain.OperationType
		amount        int64
		wantBalance   int64
		wantErr       error
		missingWallet bool
	}{
		{
			name:          "deposit",
			initial:       100,
			operationType: domain.OperationTypeDeposit,
			amount:        50,
			wantBalance:   150,
		},
		{
			name:          "withdraw",
			initial:       100,
			operationType: domain.OperationTypeWithdraw,
			amount:        40,
			wantBalance:   60,
		},
		{
			name:          "withdraw to zero",
			initial:       100,
			operationType: domain.OperationTypeWithdraw,
			amount:        100,
			wantBalance:   0,
		},
		{
			name:          "deposit to max int64",
			initial:       math.MaxInt64 - 1,
			operationType: domain.OperationTypeDeposit,
			amount:        1,
			wantBalance:   math.MaxInt64,
		},
		{
			name:          "balance overflow",
			initial:       math.MaxInt64,
			operationType: domain.OperationTypeDeposit,
			amount:        1,
			wantBalance:   math.MaxInt64,
			wantErr:       repository.ErrBalanceOverflow,
		},
		{
			name:          "insufficient funds",
			initial:       100,
			operationType: domain.OperationTypeWithdraw,
			amount:        101,
			wantBalance:   100,
			wantErr:       repository.ErrSmallBalance,
		},
		{
			name:          "wallet not found",
			operationType: domain.OperationTypeDeposit,
			amount:        100,
			wantErr:       domain.ErrWalletNotFound,
			missingWallet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id domain.WalletID

			if tt.missingWallet {
				id = newWalletID(t)
			} else {
				id = createWallet(t, ctx, repo)

				if tt.initial != 0 {
					if err := repo.UpdateBalance(
						ctx,
						withBalance(t, id, tt.initial),
					); err != nil {
						t.Fatal(err)
					}
				}
			}

			operation := withOperation(
				t,
				id,
				tt.operationType,
				tt.amount,
			)

			err := repo.ApplyOperation(ctx, operation)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"error = %v; want %v",
					err,
					tt.wantErr,
				)
			}

			if !tt.missingWallet {
				assertBalance(
					t,
					ctx,
					repo,
					id,
					tt.wantBalance,
				)
			}
		})
	}
}
