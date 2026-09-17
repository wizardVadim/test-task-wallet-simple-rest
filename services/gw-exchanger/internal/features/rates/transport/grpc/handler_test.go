package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"contracts/exchange"
	"exchanger-app/internal/core/domain"
	"exchanger-app/internal/features/rates/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubRateService struct {
	getAllFn  func(context.Context) ([]domain.ExchangeRate, error)
	getRateFn func(context.Context, domain.Currency, domain.Currency) (domain.ExchangeRate, error)
}

func (s *stubRateService) GetAll(ctx context.Context) ([]domain.ExchangeRate, error) {
	if s.getAllFn == nil {
		panic("unexpected GetAll call")
	}

	return s.getAllFn(ctx)
}

func (s *stubRateService) GetRate(
	ctx context.Context,
	from domain.Currency,
	to domain.Currency,
) (domain.ExchangeRate, error) {
	if s.getRateFn == nil {
		panic("unexpected GetRate call")
	}

	return s.getRateFn(ctx, from, to)
}

func TestHandlerGetExchangeRates(t *testing.T) {
	usd := mustCurrency(t, domain.CurrencyUSD)
	eur := mustCurrency(t, domain.CurrencyEUR)
	rub := mustCurrency(t, domain.CurrencyRUB)

	rates := []domain.ExchangeRate{
		mustExchangeRate(t, 1, usd, usd),
		mustExchangeRate(t, 0.87, usd, eur),
		mustExchangeRate(t, 90.1, usd, rub),
	}

	rateService := &stubRateService{
		getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
			return rates, nil
		},
	}

	handler := New(rateService)

	response, err := handler.GetExchangeRates(
		context.Background(),
		&exchange.Empty{},
	)
	if err != nil {
		t.Fatalf("GetExchangeRates() unexpected error: %v", err)
	}

	if response.GetBaseCurrency() != string(domain.BaseCurrency) {
		t.Errorf(
			"BaseCurrency = %q, want %q",
			response.GetBaseCurrency(),
			domain.BaseCurrency,
		)
	}

	if len(response.GetRates()) != 3 {
		t.Fatalf("Rates len = %d, want 3", len(response.GetRates()))
	}

	if response.GetRates()["USD"] != 1 {
		t.Errorf("USD rate = %v, want 1", response.GetRates()["USD"])
	}

	if response.GetRates()["EUR"] != float32(0.87) {
		t.Errorf("EUR rate = %v, want %v", response.GetRates()["EUR"], float32(0.87))
	}

	if response.GetRates()["RUB"] != float32(90.1) {
		t.Errorf("RUB rate = %v, want %v", response.GetRates()["RUB"], float32(90.1))
	}
}

func TestHandlerGetExchangeRateForCurrency(t *testing.T) {
	eur := mustCurrency(t, domain.CurrencyEUR)
	rub := mustCurrency(t, domain.CurrencyRUB)

	expected := mustExchangeRate(
		t,
		float32(90.1)/float32(0.87),
		eur,
		rub,
	)

	rateService := &stubRateService{
		getRateFn: func(
			ctx context.Context,
			from domain.Currency,
			to domain.Currency,
		) (domain.ExchangeRate, error) {
			if !from.IsEqual(eur) {
				t.Fatalf(
					"from = %q, want EUR",
					from.CurrencyType(),
				)
			}

			if !to.IsEqual(rub) {
				t.Fatalf(
					"to = %q, want RUB",
					to.CurrencyType(),
				)
			}

			return expected, nil
		},
	}

	handler := New(rateService)

	// lower case is intentional:
	// domain must normalize protobuf input.
	response, err := handler.GetExchangeRateForCurrency(
		context.Background(),
		&exchange.CurrencyRequest{
			FromCurrency: "eur",
			ToCurrency:   "rub",
		},
	)
	if err != nil {
		t.Fatalf(
			"GetExchangeRateForCurrency() unexpected error: %v",
			err,
		)
	}

	if response.GetFromCurrency() != "EUR" {
		t.Errorf(
			"FromCurrency = %q, want EUR",
			response.GetFromCurrency(),
		)
	}

	if response.GetToCurrency() != "RUB" {
		t.Errorf(
			"ToCurrency = %q, want RUB",
			response.GetToCurrency(),
		)
	}

	if response.GetRate() != expected.Rate().Value() {
		t.Errorf(
			"Rate = %v, want %v",
			response.GetRate(),
			expected.Rate().Value(),
		)
	}
}

func TestHandlerGetExchangeRateForCurrencyInvalidFrom(t *testing.T) {
	rateService := &stubRateService{
		getRateFn: func(
			ctx context.Context,
			from domain.Currency,
			to domain.Currency,
		) (domain.ExchangeRate, error) {
			t.Fatal("GetRate must not be called")
			return domain.ExchangeRate{}, nil
		},
	}

	handler := New(rateService)

	_, err := handler.GetExchangeRateForCurrency(
		context.Background(),
		&exchange.CurrencyRequest{
			FromCurrency: "GBP",
			ToCurrency:   "USD",
		},
	)

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf(
			"status code = %v, want %v, error = %v",
			status.Code(err),
			codes.InvalidArgument,
			err,
		)
	}
}

func TestHandlerGetExchangeRateForCurrencyInvalidTo(t *testing.T) {
	rateService := &stubRateService{
		getRateFn: func(
			ctx context.Context,
			from domain.Currency,
			to domain.Currency,
		) (domain.ExchangeRate, error) {
			t.Fatal("GetRate must not be called")
			return domain.ExchangeRate{}, nil
		},
	}

	handler := New(rateService)

	_, err := handler.GetExchangeRateForCurrency(
		context.Background(),
		&exchange.CurrencyRequest{
			FromCurrency: "USD",
			ToCurrency:   "GBP",
		},
	)

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf(
			"status code = %v, want %v, error = %v",
			status.Code(err),
			codes.InvalidArgument,
			err,
		)
	}
}

func TestHandlerGetExchangeRateForCurrencyNotFound(t *testing.T) {
	usd := mustCurrency(t, domain.CurrencyUSD)
	eur := mustCurrency(t, domain.CurrencyEUR)

	rateService := &stubRateService{
		getRateFn: func(
			ctx context.Context,
			from domain.Currency,
			to domain.Currency,
		) (domain.ExchangeRate, error) {
			return domain.ExchangeRate{}, service.ErrNotFoundCurrency
		},
	}

	handler := New(rateService)

	_, err := handler.GetExchangeRateForCurrency(
		context.Background(),
		&exchange.CurrencyRequest{
			FromCurrency: string(usd.CurrencyType()),
			ToCurrency:   string(eur.CurrencyType()),
		},
	)

	if status.Code(err) != codes.NotFound {
		t.Fatalf(
			"status code = %v, want %v, error = %v",
			status.Code(err),
			codes.NotFound,
			err,
		)
	}
}

func TestHandlerGetExchangeRatesInternalError(t *testing.T) {
	repositoryErr := errors.New("database connection failed")

	rateService := &stubRateService{
		getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
			return nil, repositoryErr
		},
	}

	handler := New(rateService)

	_, err := handler.GetExchangeRates(
		context.Background(),
		&exchange.Empty{},
	)

	if status.Code(err) != codes.Internal {
		t.Fatalf(
			"status code = %v, want %v, error = %v",
			status.Code(err),
			codes.Internal,
			err,
		)
	}
}

func TestHandlerCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rateService := &stubRateService{
		getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
			t.Fatal("GetAll must not be called")
			return nil, nil
		},
	}

	handler := New(rateService)

	_, err := handler.GetExchangeRates(
		ctx,
		&exchange.Empty{},
	)

	if status.Code(err) != codes.Canceled {
		t.Fatalf(
			"status code = %v, want %v, error = %v",
			status.Code(err),
			codes.Canceled,
			err,
		)
	}
}

func TestHandlerDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithCancelCause(
		context.Background(),
	)
	cancel(context.DeadlineExceeded)

	// We need an actual context whose Err() is DeadlineExceeded.
	deadlineCtx, deadlineCancel := context.WithDeadline(
		context.Background(),
		time.Now().Add(-time.Second),
	)
	defer deadlineCancel()

	_ = ctx

	rateService := &stubRateService{
		getAllFn: func(ctx context.Context) ([]domain.ExchangeRate, error) {
			t.Fatal("GetAll must not be called")
			return nil, nil
		},
	}

	handler := New(rateService)

	_, err := handler.GetExchangeRates(
		deadlineCtx,
		&exchange.Empty{},
	)

	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf(
			"status code = %v, want %v, error = %v",
			status.Code(err),
			codes.DeadlineExceeded,
			err,
		)
	}
}

func mustCurrency(
	t *testing.T,
	currencyType domain.CurrencyType,
) domain.Currency {
	t.Helper()

	currency, err := domain.NewCurrency(currencyType)
	if err != nil {
		t.Fatalf(
			"NewCurrency(%q): %v",
			currencyType,
			err,
		)
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

	exchangeRate, err := domain.NewExchangeRate(
		rate,
		from,
		to,
	)
	if err != nil {
		t.Fatalf("NewExchangeRate(): %v", err)
	}

	return exchangeRate
}
