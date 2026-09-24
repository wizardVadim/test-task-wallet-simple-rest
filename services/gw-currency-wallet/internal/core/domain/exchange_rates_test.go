package domain_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"wallet-app/internal/core/domain"
)

func TestExchangeRatesValidation(t *testing.T) {
	usd, err := domain.NewCurrency(domain.CurrencyTypeUSD)
	if err != nil {
		t.Fatal(err)
	}
	eur, err := domain.NewCurrency(domain.CurrencyTypeEUR)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		base    domain.Currency
		rates   domain.Rates
		want    error
		invalid bool
	}{
		{name: "valid", base: usd, rates: domain.Rates{"USD": 1, "EUR": 0.85, "RUB": 90}},
		{name: "invalid base", rates: domain.Rates{"USD": 1}, want: domain.ErrInvalidCurrencyType},
		{name: "non USD base", base: eur, rates: domain.Rates{"USD": 1, "EUR": 1}, want: domain.ErrBaseCurrencyNotConsists},
		{name: "nil rates", base: usd, want: domain.ErrExchangeRatesNil},
		{name: "empty rates", base: usd, rates: domain.Rates{}, invalid: true},
		{name: "missing base rate", base: usd, rates: domain.Rates{"EUR": 0.85}, invalid: true},
		{name: "invalid currency", base: usd, rates: domain.Rates{"USD": 1, "GBP": 1}, want: domain.ErrInvalidCurrencyType},
		{name: "wrong base rate", base: usd, rates: domain.Rates{"USD": 2}, want: domain.ErrInvalidBaseCurrencyRate},
		{name: "zero rate", base: usd, rates: domain.Rates{"USD": 1, "EUR": 0}, want: domain.ErrInvalidExchangeRateValue},
		{name: "negative rate", base: usd, rates: domain.Rates{"USD": 1, "EUR": -1}, want: domain.ErrInvalidExchangeRateValue},
		{name: "NaN", base: usd, rates: domain.Rates{"USD": 1, "EUR": float32(math.NaN())}, want: domain.ErrInvalidExchangeRateValue},
		{name: "positive infinity", base: usd, rates: domain.Rates{"USD": 1, "EUR": float32(math.Inf(1))}, want: domain.ErrInvalidExchangeRateValue},
		{name: "negative infinity", base: usd, rates: domain.Rates{"USD": 1, "EUR": float32(math.Inf(-1))}, want: domain.ErrInvalidExchangeRateValue},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewExchangeRates(tt.base, tt.rates)
			if tt.invalid || tt.want != nil {
				if err == nil {
					t.Fatal("expected validation error")
				}
				if tt.want != nil && !errors.Is(err, tt.want) {
					t.Fatalf("error = %v; want %v", err, tt.want)
				}
				if got.BaseCurrency() != (domain.Currency{}) || len(got.Rates()) != 0 {
					t.Error("invalid input returned populated model")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.BaseCurrency() != tt.base || !reflect.DeepEqual(got.Rates(), tt.rates) {
				t.Error("model did not preserve base and rates")
			}
		})
	}
}

func TestExchangeRatesMapIsolation(t *testing.T) {
	for _, source := range []string{"constructor input", "getter result"} {
		t.Run(source, func(t *testing.T) {
			usd, err := domain.NewCurrency(domain.CurrencyTypeUSD)
			if err != nil {
				t.Fatal(err)
			}
			input := domain.Rates{"USD": 1, "EUR": 0.85, "RUB": 90}
			rates, err := domain.NewExchangeRates(usd, input)
			if err != nil {
				t.Fatal(err)
			}
			external := input
			if source == "getter result" {
				external = rates.Rates()
			}
			external["USD"] = 2
			delete(external, "EUR")
			external["GBP"] = 1
			want := domain.Rates{"USD": 1, "EUR": 0.85, "RUB": 90}
			if !reflect.DeepEqual(rates.Rates(), want) {
				t.Errorf("external mutation changed model: %v", rates.Rates())
			}
		})
	}
}
