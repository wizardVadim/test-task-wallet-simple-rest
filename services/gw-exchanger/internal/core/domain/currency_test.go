package domain_test

import (
	"errors"
	"exchanger-app/internal/core/domain"
	"testing"
)

func TestNewCurrency(t *testing.T) {
	tests := []struct {
		name         string
		currencyType domain.CurrencyType
		wantErr      error
	}{
		{
			name:         "test new currency RUB",
			currencyType: domain.CurrencyRUB,
			wantErr:      nil,
		},
		{
			name:         "test new currency EUR",
			currencyType: domain.CurrencyEUR,
			wantErr:      nil,
		},
		{
			name:         "test new currency USD",
			currencyType: domain.CurrencyUSD,
			wantErr:      nil,
		},
		{
			name:         "test low register",
			currencyType: "usd",
			wantErr:      domain.ErrInvalidCurrencyType,
		},
		{
			name:         "test empty currency type",
			currencyType: "",
			wantErr:      domain.ErrInvalidCurrencyType,
		},
		{
			name:         "test wrong currency type",
			currencyType: "WRONG",
			wantErr:      domain.ErrInvalidCurrencyType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewCurrency(tt.currencyType)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewCurrency() unexpected error: %v", err)
			}
		})
	}
}

func mustCurrency(t *testing.T, currencyType domain.CurrencyType) domain.Currency {
	t.Helper()
	currency, err := domain.NewCurrency(currencyType)
	if err != nil {
		t.Fatalf("mustCurrency() got error %v", err)
	}
	return currency
}

func TestCurrencyType(t *testing.T) {
	tests := []struct {
		name         string
		currencyType domain.CurrencyType
		wantType     domain.CurrencyType
	}{
		{
			name:         "test get currency RUB",
			currencyType: domain.CurrencyRUB,
			wantType:     domain.CurrencyRUB,
		},
		{
			name:         "test get currency EUR",
			currencyType: domain.CurrencyEUR,
			wantType:     domain.CurrencyEUR,
		},
		{
			name:         "test get currency USD",
			currencyType: domain.CurrencyUSD,
			wantType:     domain.CurrencyUSD,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currency := mustCurrency(t, tt.currencyType)
			currencyType := currency.CurrencyType()
			if currencyType != tt.wantType {
				t.Errorf("CurrencyType() not expected value; got = %+v, want = %+v", currencyType, tt.wantType)
			}
		})
	}
}
