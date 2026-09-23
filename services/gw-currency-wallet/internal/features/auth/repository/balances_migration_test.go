package repository_test

import (
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMulticurrencyMigration(t *testing.T) {
	ctx, repo, pool := setupUsers(t)
	apply := func(name string) {
		t.Helper()
		sql, err := os.ReadFile("../../../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	// setupUsers has already applied the independent users migration in an isolated schema.
	apply("000001_create_wallets.up.sql")
	if _, err := pool.Exec(ctx, "INSERT INTO wallets (id,balance) VALUES ($1,123)", uuid.New()); err != nil {
		t.Fatal(err)
	}
	owner := user(t, uuid.New(), "owner", "owner@example.com")
	other := user(t, uuid.New(), "other", "other@example.com")
	if err := repo.Create(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	apply("000003_multicurrency_wallets.up.sql")
	var exists bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('wallets') IS NOT NULL").Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("old wallets table still exists")
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM currencies WHERE code IN ('USD','RUB','EUR')").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("seeded currency count = %d; want 3", count)
	}
	for _, currency := range []string{"USD", "RUB", "EUR"} {
		var amount int64
		if err := pool.QueryRow(ctx, "INSERT INTO balances (user_id,currency) VALUES ($1,$2) RETURNING amount", owner.ID(), currency).Scan(&amount); err != nil {
			t.Fatal(err)
		}
		if amount != 0 {
			t.Errorf("default amount = %d; want 0", amount)
		}
	}
	if _, err := pool.Exec(ctx, "INSERT INTO balances (user_id,currency,amount) VALUES ($1,'USD',9223372036854775807)", other.ID()); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, sql, code string
		args            []any
	}{
		{"duplicate balance", "INSERT INTO balances(user_id,currency) VALUES ($1,'USD')", "23505", []any{owner.ID()}},
		{"missing user", "INSERT INTO balances(user_id,currency) VALUES ($1,'USD')", "23503", []any{uuid.New()}},
		{"unknown currency", "INSERT INTO balances(user_id,currency) VALUES ($1,'GBP')", "23503", []any{other.ID()}},
		{"lowercase currency", "INSERT INTO balances(user_id,currency) VALUES ($1,'usd')", "23503", []any{other.ID()}},
		{"null user", "INSERT INTO balances(user_id,currency) VALUES (NULL,'USD')", "23502", nil},
		{"null currency", "INSERT INTO balances(user_id,currency) VALUES ($1,NULL)", "23502", []any{other.ID()}},
		{"negative amount", "INSERT INTO balances(user_id,currency,amount) VALUES ($1,'EUR',-1)", "23514", []any{other.ID()}},
		{"null amount", "INSERT INTO balances(user_id,currency,amount) VALUES ($1,'EUR',NULL)", "23502", []any{other.ID()}},
		{"negative update", "UPDATE balances SET amount=-1 WHERE user_id=$1 AND currency='USD'", "23514", []any{other.ID()}},
		{"referenced currency deletion", "DELETE FROM currencies WHERE code='USD'", "23503", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.sql, tt.args...)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tt.code {
				t.Fatalf("error = %v; want SQLSTATE %s", err, tt.code)
			}
		})
	}
	if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id=$1", owner.ID()); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM balances WHERE user_id=$1", owner.ID()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("user deletion did not cascade to balances")
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM balances WHERE user_id=$1", other.ID()).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("user deletion affected another user's balance")
	}
	apply("000003_multicurrency_wallets.down.sql")
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM wallets").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("rollback should restore an empty wallets table")
	}
	if err := pool.QueryRow(ctx, "SELECT to_regclass('balances') IS NULL AND to_regclass('currencies') IS NULL").Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("rollback left new tables behind")
	}
	var amount int64
	if err := pool.QueryRow(ctx, "INSERT INTO wallets(id) VALUES ($1) RETURNING balance", uuid.New()).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 0 {
		t.Fatal("rollback lost default balance")
	}
	if _, err := repo.GetByUsername(ctx, "other"); err != nil {
		t.Fatal("rollback affected remaining user:", err)
	}
	apply("000003_multicurrency_wallets.up.sql")
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM balances").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("reapplied migration should create empty balances")
	}
}
