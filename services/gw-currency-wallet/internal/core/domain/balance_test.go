package domain_test

import (
	"errors"
	"github.com/google/uuid"
	"math"
	"testing"
	"wallet-app/internal/core/domain"
)

func TestCurrency(t *testing.T) {
	for _, code := range []domain.CurrencyType{domain.CurrencyTypeUSD, domain.CurrencyTypeRUB, domain.CurrencyTypeEUR} {
		t.Run(string(code), func(t *testing.T) {
			got, err := domain.NewCurrency(code)
			if err != nil {
				t.Fatal(err)
			}
			if got.CurrencyType() != code {
				t.Errorf("currency = %q; want %q", got.CurrencyType(), code)
			}
		})
	}
	for _, code := range []domain.CurrencyType{"", "GBP", "usd", " USD ", "US"} {
		t.Run("invalid_"+string(code), func(t *testing.T) {
			got, err := domain.NewCurrency(code)
			if !errors.Is(err, domain.ErrInvalidCurrencyType) {
				t.Errorf("error = %v; want ErrInvalidCurrencyType", err)
			}
			if got != (domain.Currency{}) {
				t.Error("invalid currency returned nonzero value")
			}
		})
	}
}

func TestBalance(t *testing.T) {
	id := uuid.MustParse("e373039e-7060-4525-90bc-d5c7d7525a80")
	currency, err := domain.NewCurrency(domain.CurrencyTypeUSD)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name     string
		id       uuid.UUID
		currency domain.Currency
		amount   int64
		wantErr  error
	}{
		{"zero", id, currency, 0, nil},
		{"positive", id, currency, 123, nil},
		{"maximum int64", id, currency, math.MaxInt64, nil},
		{"nil user", uuid.Nil, currency, 0, domain.ErrInvalidUserID},
		{"zero currency", id, domain.Currency{}, 0, domain.ErrInvalidCurrencyType},
		{"negative", id, currency, -1, domain.ErrInvalidBalanceAmount},
		{"minimum int64", id, currency, math.MinInt64, domain.ErrInvalidBalanceAmount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewBalance(tt.id, tt.currency, tt.amount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v; want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got != (domain.Balance{}) {
					t.Error("invalid balance returned nonzero value")
				}
				return
			}
			if got.UserID() != tt.id || got.Currency() != tt.currency || got.Amount() != tt.amount {
				t.Error("balance getters do not match input")
			}
		})
	}
}
