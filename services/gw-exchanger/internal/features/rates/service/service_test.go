package service

import (
	"context"
	"errors"
	"exchanger-app/internal/core/domain"
	"testing"
)

type stubRateRepository struct {
	getAllFn func(ctx context.Context) ([]domain.ExchangeRate, error)
}

func (s *stubRateRepository) GetAll(ctx context.Context) ([]domain.ExchangeRate, error) {
	return s.getAllFn(ctx)
}

func TestServiceGetRate(t *testing.T) {
	usd := mustCurrency(t, domain.CurrencyUSD)
	eur := mustCurrency(t, domain.CurrencyEUR)
	rub := mustCurrency(t, domain.CurrencyRUB)

	baseRates := []domain.ExchangeRate{
		mustExchangeRate(t, 1, usd, usd),
		mustExchangeRate(t, 0.87, usd, eur),
		mustExchangeRate(t, 90.1, usd, rub),
	}

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name       string
		ctx        context.Context
		from       domain.Currency
		to         domain.Currency
		repository RateRepository
		wantRate   float32
		wantErr    error
	}{
		{
			name: "USD to EUR",
			ctx:  context.Background(),
			from: usd,
			to:   eur,
			repository: &stubRateRepository{
				getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
					return baseRates, nil
				},
			},
			wantRate: 0.87,
		},
		{
			name: "EUR to RUB cross rate",
			ctx:  context.Background(),
			from: eur,
			to:   rub,
			repository: &stubRateRepository{
				getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
					return baseRates, nil
				},
			},
			wantRate: 90.1 / 0.87,
		},
		{
			name: "from currency not found",
			ctx:  context.Background(),
			from: eur,
			to:   rub,
			repository: &stubRateRepository{
				getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
					return []domain.ExchangeRate{
						mustExchangeRate(t, 1, usd, usd),
						mustExchangeRate(t, 90.1, usd, rub),
					}, nil
				},
			},
			wantErr: ErrNotFoundCurrency,
		},
		{
			name: "to currency not found",
			ctx:  context.Background(),
			from: usd,
			to:   eur,
			repository: &stubRateRepository{
				getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
					return []domain.ExchangeRate{
						mustExchangeRate(t, 1, usd, usd),
					}, nil
				},
			},
			wantErr: ErrNotFoundCurrency,
		},
		{
			name: "repository error",
			ctx:  context.Background(),
			from: usd,
			to:   eur,
			repository: &stubRateRepository{
				getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
					return nil, repositoryErr
				},
			},
			wantErr: repositoryErr,
		},
		{
			name: "invalid from currency",
			ctx:  context.Background(),
			from: domain.Currency{},
			to:   eur,
			repository: &stubRateRepository{
				getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
					t.Fatal("repository must not be called")
					return nil, nil
				},
			},
			wantErr: domain.ErrInvalidCurrencyType,
		},
		{
			name: "invalid to currency",
			ctx:  context.Background(),
			from: usd,
			to:   domain.Currency{},
			repository: &stubRateRepository{
				getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
					t.Fatal("repository must not be called")
					return nil, nil
				},
			},
			wantErr: domain.ErrInvalidCurrencyType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.repository)

			got, err := s.GetRate(tt.ctx, tt.from, tt.to)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("GetRate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetRate() unexpected error: %v", err)
			}

			if got.FromCurrency().CurrencyType() != tt.from.CurrencyType() {
				t.Errorf(
					"FromCurrency() = %v, want %v",
					got.FromCurrency().CurrencyType(),
					tt.from.CurrencyType(),
				)
			}

			if got.ToCurrency().CurrencyType() != tt.to.CurrencyType() {
				t.Errorf(
					"ToCurrency() = %v, want %v",
					got.ToCurrency().CurrencyType(),
					tt.to.CurrencyType(),
				)
			}

			if got.Rate().Value() != tt.wantRate {
				t.Errorf(
					"Rate().Value() = %v, want %v",
					got.Rate().Value(),
					tt.wantRate,
				)
			}
		})
	}
}

func TestServiceGetRateCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := &stubRateRepository{
		getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
			t.Fatal("repository must not be called")
			return nil, nil
		},
	}

	s := New(repository)

	_, err := s.GetRate(
		ctx,
		mustCurrency(t, domain.CurrencyUSD),
		mustCurrency(t, domain.CurrencyEUR),
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetRate() error = %v, want %v", err, context.Canceled)
	}
}

func TestServiceGetAll(t *testing.T) {
	usd := mustCurrency(t, domain.CurrencyUSD)
	eur := mustCurrency(t, domain.CurrencyEUR)

	want := []domain.ExchangeRate{
		mustExchangeRate(t, 1, usd, usd),
		mustExchangeRate(t, 0.87, usd, eur),
	}

	repository := &stubRateRepository{
		getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
			return want, nil
		},
	}

	s := New(repository)

	got, err := s.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll() unexpected error: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("GetAll() len = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i].FromCurrency().CurrencyType() != want[i].FromCurrency().CurrencyType() {
			t.Errorf("item %d: wrong from currency", i)
		}

		if got[i].ToCurrency().CurrencyType() != want[i].ToCurrency().CurrencyType() {
			t.Errorf("item %d: wrong to currency", i)
		}

		if got[i].Rate().Value() != want[i].Rate().Value() {
			t.Errorf(
				"item %d: rate = %v, want %v",
				i,
				got[i].Rate().Value(),
				want[i].Rate().Value(),
			)
		}
	}
}

func TestServiceGetAllCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repository := &stubRateRepository{
		getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
			t.Fatal("repository must not be called")
			return nil, nil
		},
	}

	s := New(repository)

	_, err := s.GetAll(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetAll() error = %v, want %v", err, context.Canceled)
	}
}

func mustCurrency(t *testing.T, currencyType domain.CurrencyType) domain.Currency {
	t.Helper()

	currency, err := domain.NewCurrency(currencyType)
	if err != nil {
		t.Fatalf("NewCurrency(%q): %v", currencyType, err)
	}

	return currency
}

func mustExchangeRate(
	t *testing.T,
	value float32,
	from domain.Currency,
	to domain.Currency,
) domain.ExchangeRate {
	t.Helper()

	rate, err := domain.NewRate(value)
	if err != nil {
		t.Fatalf("NewRate(%v): %v", value, err)
	}

	exchangeRate, err := domain.NewExchangeRate(rate, from, to)
	if err != nil {
		t.Fatalf("NewExchangeRate(): %v", err)
	}

	return exchangeRate
}
