package repository_test

import (
	"context"
	"errors"
	"testing"

	"exchanger-app/internal/core/domain"
	"exchanger-app/internal/features/rates/repository"
	"exchanger-app/internal/features/rates/service"

	"github.com/jackc/pgx/v5"
)

var _ service.RateRepository = (*repository.PostgresRepository)(nil)

type rateRow struct {
	code  string
	value float32
}

type stubRows struct {
	pgx.Rows
	data        []rateRow
	index       int
	closed      bool
	scanErr     error
	terminalErr error
}

func (r *stubRows) Next() bool {
	if r.index >= len(r.data) {
		return false
	}
	r.index++
	return true
}
func (r *stubRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.index-1]
	*dest[0].(*string) = row.code
	*dest[1].(*float32) = row.value
	return nil
}
func (r *stubRows) Close()     { r.closed = true }
func (r *stubRows) Err() error { return r.terminalErr }

type stubDB struct {
	rows  *stubRows
	err   error
	calls int
	ctx   context.Context
}

func (db *stubDB) Query(ctx context.Context, _ string, _ ...any) (pgx.Rows, error) {
	db.calls++
	db.ctx = ctx
	return db.rows, db.err
}

func TestGetAllMapsRates(t *testing.T) {
	rows := &stubRows{data: []rateRow{{"EUR", 0.87}, {"USD", 1}, {"RUB", 90.1}}}
	db := &stubDB{rows: rows}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got, err := repository.NewPostgresRepository(db).GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(rows.data) {
		t.Fatalf("got %d rates, want %d", len(got), len(rows.data))
	}
	want := map[domain.CurrencyType]float32{domain.CurrencyEUR: 0.87, domain.CurrencyUSD: 1, domain.CurrencyRUB: 90.1}
	for _, rate := range got {
		if rate.FromCurrency().CurrencyType() != domain.CurrencyUSD {
			t.Errorf("source currency = %s, want USD", rate.FromCurrency().CurrencyType())
		}
		code := rate.ToCurrency().CurrencyType()
		value, ok := want[code]
		if !ok || rate.Rate().Value() != value {
			t.Errorf("unexpected rate %s = %v", code, rate.Rate().Value())
		}
		delete(want, code)
	}
	if len(want) != 0 {
		t.Errorf("missing currencies: %v", want)
	}
	if !rows.closed {
		t.Error("rows were not closed")
	}
	if db.calls != 1 || db.ctx != ctx {
		t.Error("expected one query with caller context")
	}
}

func TestGetAllEmpty(t *testing.T) {
	rows := &stubRows{}
	got, err := repository.NewPostgresRepository(&stubDB{rows: rows}).GetAll(context.Background())
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want empty result without error", got, err)
	}
	if !rows.closed {
		t.Error("rows were not closed")
	}
}

func TestGetAllCanceledBeforeQuery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db := &stubDB{}
	got, err := repository.NewPostgresRepository(db).GetAll(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if len(got) != 0 || db.calls != 0 {
		t.Error("canceled call must not query or return rates")
	}
}

func TestGetAllQueryError(t *testing.T) {
	wantErr := errors.New("query failed")
	got, err := repository.NewPostgresRepository(&stubDB{err: wantErr}).GetAll(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("got %v, want wrapped query error", err)
	}
	if len(got) != 0 {
		t.Error("returned rates after query failure")
	}
}

func TestGetAllRowErrors(t *testing.T) {
	scanErr := errors.New("scan failed")
	tests := []struct {
		name    string
		rows    stubRows
		wantErr error
	}{
		{name: "scan failure", rows: stubRows{data: []rateRow{{"RUB", 90}}, scanErr: scanErr}, wantErr: scanErr},
		{name: "iteration canceled after a valid row", rows: stubRows{data: []rateRow{{"RUB", 90}}, terminalErr: context.Canceled}, wantErr: context.Canceled},
		{name: "unsupported currency after a valid row", rows: stubRows{data: []rateRow{{"RUB", 90}, {"GBP", 0.8}}}, wantErr: domain.ErrInvalidCurrencyType},
		{name: "invalid rate", rows: stubRows{data: []rateRow{{"RUB", 0}}}, wantErr: domain.ErrRateValueBelowEqualsZero},
		{name: "invalid USD rate", rows: stubRows{data: []rateRow{{"USD", 2}}}, wantErr: domain.ErrInvalidRateValueForSameCurrency},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repository.NewPostgresRepository(&stubDB{rows: &tt.rows}).GetAll(context.Background())
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got error %v, want %v", err, tt.wantErr)
			}
			if len(got) != 0 {
				t.Errorf("returned partial result on failure: %v", got)
			}
			if !tt.rows.closed {
				t.Error("rows were not closed on failure")
			}
		})
	}
}
