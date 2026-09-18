package domain_test

import (
	"errors"
	"exchanger-app/internal/core/domain"
	"math"
	"testing"
)

func TestNewRate(t *testing.T) {
	tests := []struct {
		name    string
		value   float32
		wantErr error
	}{
		{
			name:    "succesful test",
			value:   2.0,
			wantErr: nil,
		},
		{
			name:    "succesful floating point test",
			value:   2.55,
			wantErr: nil,
		},
		{
			name:    "zero test",
			value:   0.0,
			wantErr: domain.ErrRateValueBelowEqualsZero,
		},
		{
			name:    "below zero test",
			value:   -2.0,
			wantErr: domain.ErrRateValueBelowEqualsZero,
		},
		{
			name:    "NaN test",
			value:   float32(math.NaN()),
			wantErr: domain.ErrRateValueIsNaN,
		},
		{
			name:    "negative inf test",
			value:   float32(math.Inf(-1)),
			wantErr: domain.ErrRateValueIsInf,
		},
		{
			name:    "positive inf test",
			value:   float32(math.Inf(1)),
			wantErr: domain.ErrRateValueIsInf,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewRate(tt.value)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewRate() unexpected error: %v", err)
			}
		})
	}
}

func mustRate(t *testing.T, value float32) domain.Rate {
	t.Helper()
	rate, err := domain.NewRate(value)
	if err != nil {
		t.Fatalf("mustRate error: %v", err)
	}
	return rate
}

func TestRateValue(t *testing.T) {
	tests := []struct {
		name  string
		value float32
	}{
		{
			name:  "simple",
			value: 1.0,
		},
		{
			name:  "floating point",
			value: 1.102,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := mustRate(t, tt.value)
			if rate.Value() != tt.value {
				t.Errorf("Rate Value() unexpected value: got = %v, want = %v", rate.Value(), tt.value)
			}
		})
	}
}

func TestNewExchangeRate(t *testing.T) {
	tests := []struct {
		name         string
		fromCurrency domain.Currency
		toCurrency   domain.Currency
		rate         domain.Rate
		wantErr      error
	}{
		{
			name:         "USD -> RUB",
			fromCurrency: mustCurrency(t, domain.CurrencyUSD),
			toCurrency:   mustCurrency(t, domain.CurrencyRUB),
			rate:         mustRate(t, 10.5),
			wantErr:      nil,
		},
		{
			name:         "RUB -> EUR",
			fromCurrency: mustCurrency(t, domain.CurrencyRUB),
			toCurrency:   mustCurrency(t, domain.CurrencyEUR),
			rate:         mustRate(t, 10.5),
			wantErr:      nil,
		},
		{
			name:         "EUR -> USD",
			fromCurrency: mustCurrency(t, domain.CurrencyEUR),
			toCurrency:   mustCurrency(t, domain.CurrencyUSD),
			rate:         mustRate(t, 10.5),
			wantErr:      nil,
		},
		{
			name:         "USD -> USD the same currency",
			fromCurrency: mustCurrency(t, domain.CurrencyUSD),
			toCurrency:   mustCurrency(t, domain.CurrencyUSD),
			rate:         mustRate(t, 1),
			wantErr:      nil,
		},
		{
			name:         "USD -> USD the same currency wrong",
			fromCurrency: mustCurrency(t, domain.CurrencyUSD),
			toCurrency:   mustCurrency(t, domain.CurrencyUSD),
			rate:         mustRate(t, 11.1),
			wantErr:      domain.ErrInvalidRateValueForSameCurrency,
		},
		{
			name:         "Empty rate",
			fromCurrency: mustCurrency(t, domain.CurrencyUSD),
			toCurrency:   mustCurrency(t, domain.CurrencyRUB),
			rate:         domain.Rate{},
			wantErr:      domain.ErrRateValueBelowEqualsZero,
		},
		{
			name:         "Empty from currency",
			fromCurrency: domain.Currency{},
			toCurrency:   mustCurrency(t, domain.CurrencyRUB),
			rate:         mustRate(t, 11.1),
			wantErr:      domain.ErrInvalidCurrencyType,
		},
		{
			name:         "Empty to currency",
			fromCurrency: mustCurrency(t, domain.CurrencyUSD),
			toCurrency:   domain.Currency{},
			rate:         mustRate(t, 11.1),
			wantErr:      domain.ErrInvalidCurrencyType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewExchangeRate(tt.rate, tt.fromCurrency, tt.toCurrency)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewExchangeRate() unexpected error: %v", err)
			}
		})
	}
}

func TestExchangeRateRate(t *testing.T) {
	fromCurrency := mustCurrency(t, domain.CurrencyEUR)
	toCurrency := mustCurrency(t, domain.CurrencyUSD)
	tests := []struct {
		name  string
		value domain.Rate
	}{
		{
			name:  "simple",
			value: mustRate(t, 10.0),
		},
		{
			name:  "floating point",
			value: mustRate(t, 10.033),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exchangeRate := mustExchangeRate(t, fromCurrency, toCurrency, tt.value)
			if exchangeRate.Rate() != tt.value {
				t.Errorf("ExchangeRate Rate() unexpected value; got = %+v, want = %+v", exchangeRate.Rate(), tt.value)
			}
		})
	}
}

func TestExchangeRateFromCurrency(t *testing.T) {
	rate := mustRate(t, 10.0)
	tests := []struct {
		name  string
		value domain.Currency
	}{
		{
			name:  "USD test",
			value: mustCurrency(t, "USD"),
		},
		{
			name:  "RUB test",
			value: mustCurrency(t, "RUB"),
		},
		{
			name:  "EUR test",
			value: mustCurrency(t, "EUR"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var toCurrency domain.Currency
			switch tt.value.CurrencyType() {
			case "USD":
				toCurrency = mustCurrency(t, domain.CurrencyEUR)
			default:
				toCurrency = mustCurrency(t, domain.CurrencyUSD)
			}
			exchangeRate := mustExchangeRate(t, tt.value, toCurrency, rate)
			if exchangeRate.FromCurrency().CurrencyType() != tt.value.CurrencyType() {
				t.Errorf("ExchangeRate FromCurrency() unexpected value; got = %+v, want = %+v", exchangeRate.FromCurrency(), tt.value)
			}
		})
	}
}

func mustExchangeRate(t *testing.T, from domain.Currency, to domain.Currency, rate domain.Rate) domain.ExchangeRate {
	t.Helper()
	exchangeRate, err := domain.NewExchangeRate(rate, from, to)
	if err != nil {
		t.Fatalf("mustExchangeRate error: %v", err)
	}
	return exchangeRate
}

func TestExchangeRateToCurrency(t *testing.T) {
	rate := mustRate(t, 10.0)
	tests := []struct {
		name  string
		value domain.Currency
	}{
		{
			name:  "USD test",
			value: mustCurrency(t, "USD"),
		},
		{
			name:  "RUB test",
			value: mustCurrency(t, "RUB"),
		},
		{
			name:  "EUR test",
			value: mustCurrency(t, "EUR"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fromCurrency domain.Currency
			switch tt.value.CurrencyType() {
			case "USD":
				fromCurrency = mustCurrency(t, domain.CurrencyEUR)
			default:
				fromCurrency = mustCurrency(t, domain.CurrencyUSD)
			}
			exchangeRate := mustExchangeRate(t, fromCurrency, tt.value, rate)
			if exchangeRate.ToCurrency().CurrencyType() != tt.value.CurrencyType() {
				t.Errorf("ExchangeRate ToCurrency() unexpected value; got = %+v, want = %+v", exchangeRate.ToCurrency(), tt.value)
			}
		})
	}
}
