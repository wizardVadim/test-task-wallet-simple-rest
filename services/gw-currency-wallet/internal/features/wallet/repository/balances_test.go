package repository_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/wallet/repository"
)

type balancesDB struct {
	repository.DBTX
	query func(context.Context, string, ...any) (pgx.Rows, error)
}

func (d balancesDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return d.query(ctx, sql, args...)
}

type balanceRow struct {
	id, currency string
	amount       int64
}
type balancesRows struct {
	pgx.Rows
	data                  []balanceRow
	index                 int
	scanErr, iterationErr error
	closed                bool
}

func (r *balancesRows) Next() bool { r.index++; return r.index <= len(r.data) }
func (r *balancesRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.index-1]
	*dest[0].(*string) = row.id
	*dest[1].(*string) = row.currency
	*dest[2].(*int64) = row.amount
	return nil
}
func (r *balancesRows) Close()     { r.closed = true }
func (r *balancesRows) Err() error { return r.iterationErr }

func TestGetBalancesMapping(t *testing.T) {
	id := uuid.New()
	for _, empty := range []bool{false, true} {
		name := "three currencies"
		if empty {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			rows := &balancesRows{}
			if !empty {
				rows.data = []balanceRow{{id.String(), "USD", 12345}, {id.String(), "RUB", 0}, {id.String(), "EUR", math.MaxInt64}}
			}
			ctx := t.Context()
			calls := 0
			db := balancesDB{query: func(gotCtx context.Context, _ string, args ...any) (pgx.Rows, error) {
				calls++
				if gotCtx != ctx || len(args) != 1 || args[0] != id {
					t.Error("incorrect query context or user ID")
				}
				return rows, nil
			}}
			got, err := repository.NewPostgresRepository(db).GetBalances(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(rows.data) {
				t.Fatalf("balance count = %d; want %d", len(got), len(rows.data))
			}
			for i, want := range rows.data {
				if got[i].UserID() != id || string(got[i].Currency().CurrencyType()) != want.currency || got[i].Amount() != want.amount {
					t.Errorf("incorrect balance %d", i)
				}
			}
			if calls != 1 || !rows.closed {
				t.Error("query not called once or rows not closed")
			}
		})
	}
}

func TestGetBalancesFailures(t *testing.T) {
	id := uuid.New()
	scanErr := errors.New("cannot scan amount")
	iterationErr := errors.New("connection lost during iteration")
	queryErr := errors.New("query failed")
	for _, tt := range []struct {
		name                                     string
		data                                     []balanceRow
		scanErr, iterationErr, queryErr, wantErr error
		canceled                                 bool
	}{
		{name: "scan error", data: []balanceRow{{id.String(), "USD", 1}}, scanErr: scanErr, wantErr: scanErr},
		{name: "iteration error discards partial results", data: []balanceRow{{id.String(), "USD", 1}}, iterationErr: iterationErr, wantErr: iterationErr},
		{name: "query error", queryErr: queryErr, wantErr: queryErr},
		{name: "canceled before query", canceled: true, wantErr: context.Canceled},
		{name: "malformed user ID", data: []balanceRow{{"invalid", "USD", 1}}, wantErr: domain.ErrInvalidUserID},
		{name: "nil user ID", data: []balanceRow{{uuid.Nil.String(), "USD", 1}}, wantErr: domain.ErrInvalidUserID},
		{name: "invalid currency", data: []balanceRow{{id.String(), "GBP", 1}}, wantErr: domain.ErrInvalidCurrencyType},
		{name: "negative stored amount", data: []balanceRow{{id.String(), "USD", -1}}, wantErr: domain.ErrInvalidBalanceAmount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			rows := &balancesRows{data: tt.data, scanErr: tt.scanErr, iterationErr: tt.iterationErr}
			calls := 0
			db := balancesDB{query: func(context.Context, string, ...any) (pgx.Rows, error) { calls++; return rows, tt.queryErr }}
			got, err := repository.NewPostgresRepository(db).GetBalances(ctx, id)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v; want %v", err, tt.wantErr)
			}
			if len(got) != 0 {
				t.Error("error returned partial balances")
			}
			if tt.canceled {
				if calls != 0 {
					t.Error("query called after cancellation")
				}
			} else if calls != 1 {
				t.Error("expected one query")
			}
			if !tt.canceled && tt.queryErr == nil && !rows.closed {
				t.Error("rows not closed")
			}
		})
	}
}
