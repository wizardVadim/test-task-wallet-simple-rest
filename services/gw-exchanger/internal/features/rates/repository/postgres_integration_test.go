package repository_test

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"exchanger-app/internal/core/domain"
	"exchanger-app/internal/features/rates/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresFixture struct {
	ctx  context.Context
	pool *pgxpool.Pool
	repo *repository.PostgresRepository
}

func newPostgresFixture(t *testing.T) postgresFixture {
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
	schema := pgx.Identifier{"exchanger_test_" + rand.Text()}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("cleanup schema: %v", err)
		}
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	// A single connection also exposes leaked rows on an early error return.
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	f := postgresFixture{ctx, pool, repository.NewPostgresRepository(pool)}
	f.migrate(t, "000001_create_exchange_rates.up.sql")
	return f
}

func (f postgresFixture) migrate(t *testing.T, name string) {
	t.Helper()
	sql, err := os.ReadFile(filepath.Join("../../../../migrations", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(f.ctx, string(sql)); err != nil {
		t.Fatalf("migration %s: %v", name, err)
	}
}

func checkSeedRates(rates []domain.ExchangeRate) error {
	want := map[domain.CurrencyType]float32{domain.CurrencyUSD: 1, domain.CurrencyRUB: 90.1, domain.CurrencyEUR: 0.87}
	if len(rates) != len(want) {
		return errors.New("unexpected number of rates")
	}
	for _, rate := range rates {
		code := rate.ToCurrency().CurrencyType()
		expected, ok := want[code]
		if !ok || rate.FromCurrency().CurrencyType() != domain.CurrencyUSD || rate.Rate().Value() != expected {
			return errors.New("unexpected currency pair or rate")
		}
		delete(want, code)
	}
	if len(want) != 0 {
		return errors.New("missing seeded currency")
	}
	return nil
}

func TestPostgresGetAll(t *testing.T) {
	f := newPostgresFixture(t)
	rates, err := f.repo.GetAll(f.ctx)
	if err != nil || len(rates) != 0 {
		t.Fatalf("empty table: %v, %v", rates, err)
	}
	f.migrate(t, "000002_seed_exchange_rates.up.sql")
	rates, err = f.repo.GetAll(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkSeedRates(rates); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresConcurrentReads(t *testing.T) {
	f := newPostgresFixture(t)
	f.migrate(t, "000002_seed_exchange_rates.up.sql")
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rates, err := f.repo.GetAll(f.ctx)
			if err == nil {
				err = checkSeedRates(rates)
			}
			if err != nil {
				t.Errorf("concurrent GetAll: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestPostgresInvalidStoredCurrency(t *testing.T) {
	f := newPostgresFixture(t)
	f.migrate(t, "000002_seed_exchange_rates.up.sql")
	if _, err := f.pool.Exec(f.ctx, "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('GBP', 0.8)"); err != nil {
		t.Fatal(err)
	}
	rates, err := f.repo.GetAll(f.ctx)
	if !errors.Is(err, domain.ErrInvalidCurrencyType) || len(rates) != 0 {
		t.Fatalf("got %v, %v; want no partial result and invalid currency", rates, err)
	}
	// The connection must still be usable after domain validation fails.
	if _, err := f.pool.Exec(f.ctx, "DELETE FROM exchange_rates WHERE code = 'GBP'"); err != nil {
		t.Fatal(err)
	}
	rates, err = f.repo.GetAll(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkSeedRates(rates); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresCancellationAndQueryFailure(t *testing.T) {
	f := newPostgresFixture(t)
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if _, err := f.repo.GetAll(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled GetAll: %v", err)
	}
	f.migrate(t, "000001_create_exchange_rates.down.sql")
	rates, err := f.repo.GetAll(f.ctx)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42P01" || len(rates) != 0 {
		t.Fatalf("missing table: %v, %v", rates, err)
	}
}

func TestPostgresRateConstraints(t *testing.T) {
	f := newPostgresFixture(t)
	for _, tt := range []struct{ name, sql, code string }{
		{"zero", "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('RUB',0)", "23514"},
		{"negative", "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('RUB',-1)", "23514"},
		{"NaN", "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('RUB','NaN')", "23514"},
		{"positive infinity", "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('RUB','Infinity')", "23514"},
		{"negative infinity", "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('RUB','-Infinity')", "23514"},
		{"USD must equal one", "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('USD',2)", "23514"},
		{"null currency", "INSERT INTO exchange_rates (code, units_per_usd) VALUES (NULL,1)", "23502"},
		{"null rate", "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('RUB',NULL)", "23502"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.pool.Exec(f.ctx, tt.sql)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != tt.code {
				t.Fatalf("got %v, want SQLSTATE %s", err, tt.code)
			}
		})
	}
	f.migrate(t, "000002_seed_exchange_rates.up.sql")
	_, err := f.pool.Exec(f.ctx, "INSERT INTO exchange_rates (code, units_per_usd) VALUES ('RUB',100)")
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("duplicate currency: %v", err)
	}
}

func TestPostgresMigrationsRoundTrip(t *testing.T) {
	f := newPostgresFixture(t)
	f.migrate(t, "000002_seed_exchange_rates.up.sql")
	f.migrate(t, "000002_seed_exchange_rates.down.sql")
	rates, err := f.repo.GetAll(f.ctx)
	if err != nil || len(rates) != 0 {
		t.Fatalf("seed rollback: %v, %v", rates, err)
	}
	f.migrate(t, "000001_create_exchange_rates.down.sql")
	var absent bool
	if err := f.pool.QueryRow(f.ctx, "SELECT to_regclass('exchange_rates') IS NULL").Scan(&absent); err != nil || !absent {
		t.Fatalf("table rollback: absent=%v, err=%v", absent, err)
	}
	f.migrate(t, "000001_create_exchange_rates.up.sql")
	f.migrate(t, "000002_seed_exchange_rates.up.sql")
	rates, err = f.repo.GetAll(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkSeedRates(rates); err != nil {
		t.Fatal(err)
	}
}
