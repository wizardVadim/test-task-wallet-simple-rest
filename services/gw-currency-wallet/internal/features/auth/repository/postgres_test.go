package repository_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/auth/repository"
	"wallet-app/internal/features/auth/service"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ service.UserRepository = (*repository.PostgresRepository)(nil)

func setupUsers(t *testing.T) (context.Context, *repository.PostgresRepository, *pgxpool.Pool) {
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
	schema := pgx.Identifier{"auth_test_" + uuid.NewString()}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 4
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	migration, err := os.ReadFile("../../../../migrations/000002_create_users.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	return ctx, repository.NewPostgresRepository(pool), pool
}

func user(t *testing.T, id uuid.UUID, username, email string) domain.User {
	t.Helper()
	u, err := domain.NewUser(id, username, email, "$test$CaseSensitiveHash")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestUserCreateAndRead(t *testing.T) {
	ctx, repo, _ := setupUsers(t)
	want := user(t, uuid.New(), " IvAn ", "Ivan@example.com")
	if err := repo.Create(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByUsername(ctx, "ivan")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Error("stored user fields differ from original")
	}
	if _, err := repo.GetByUsername(ctx, "missing"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("missing user: %v", err)
	}
}

func TestUserUniqueConstraints(t *testing.T) {
	ctx, repo, pool := setupUsers(t)
	original := user(t, uuid.New(), "ivan", "Ivan@example.com")
	if err := repo.Create(ctx, original); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		u    domain.User
		want error
	}{
		{"normalized username", user(t, uuid.New(), " IVAN ", "different@example.com"), domain.ErrUsernameAlreadyExists},
		{"email exact", user(t, uuid.New(), "other", "Ivan@example.com"), domain.ErrEmailAlreadyExists},
		{"email case insensitive", user(t, uuid.New(), "another", "ivan@EXAMPLE.COM"), domain.ErrEmailAlreadyExists},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := repo.Create(ctx, tt.u); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
	duplicateID := user(t, original.ID(), "distinct", "distinct@example.com")
	err := repo.Create(ctx, duplicateID)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "users_pkey" {
		t.Fatalf("primary key error not preserved: %v", err)
	}
	if errors.Is(err, domain.ErrUsernameAlreadyExists) || errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatal("primary key conflict misclassified")
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d, error=%v", count, err)
	}
}

func TestConcurrentUserEmailRegistration(t *testing.T) {
	ctx, repo, pool := setupUsers(t)
	const n = 8
	users := make([]domain.User, n)
	for i := range users {
		email := "User@example.com"
		if i%2 == 0 {
			email = "user@EXAMPLE.COM"
		}
		users[i] = user(t, uuid.New(), fmt.Sprintf("user%d", i), email)
	}
	start := make(chan struct{})
	results := make(chan error, n)
	var wg sync.WaitGroup
	for _, u := range users {
		wg.Add(1)
		go func(u domain.User) { defer wg.Done(); <-start; results <- repo.Create(ctx, u) }(u)
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, domain.ErrEmailAlreadyExists) {
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Errorf("successful registrations=%d, want 1", successes)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d, error=%v", count, err)
	}
}

func TestUsersCancellationAndRollback(t *testing.T) {
	ctx, repo, pool := setupUsers(t)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := repo.Create(canceled, user(t, uuid.New(), "ivan", "ivan@example.com")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled create: %v", err)
	}
	if _, err := repo.GetByUsername(canceled, "ivan"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read: %v", err)
	}
	down, err := os.ReadFile("../../../../migrations/000002_create_users.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetByUsername(ctx, "ivan")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42P01" {
		t.Fatalf("table error must not become user-not-found: %v", err)
	}
}
