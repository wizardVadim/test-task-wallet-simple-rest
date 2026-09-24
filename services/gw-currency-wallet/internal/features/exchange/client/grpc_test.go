package client_test

import (
	"context"
	"contracts/exchange"
	"errors"
	"math"
	"reflect"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/exchange/client"
	"wallet-app/internal/features/exchange/service"
)

var _ service.RatesProvider = (*client.Client)(nil)

type exchangeStub struct {
	exchange.ExchangeServiceClient
	get func(context.Context, *exchange.Empty) (*exchange.ExchangeRatesResponse, error)
}

func (s exchangeStub) GetExchangeRates(ctx context.Context, req *exchange.Empty, _ ...grpc.CallOption) (*exchange.ExchangeRatesResponse, error) {
	return s.get(ctx, req)
}

func TestGetExchangeRates(t *testing.T) {
	rpcErr := status.Error(codes.Unavailable, "exchanger unavailable")
	for _, tt := range []struct {
		name, base   string
		rates        map[string]float32
		rpcErr, want error
		invalid      bool
	}{
		{name: "success", base: "USD", rates: map[string]float32{"USD": 1, "EUR": 0.85, "RUB": 90}},
		{name: "RPC failure", rpcErr: rpcErr, want: rpcErr},
		{name: "cancellation", rpcErr: context.Canceled, want: context.Canceled},
		{name: "deadline", rpcErr: status.Error(codes.DeadlineExceeded, "timeout"), invalid: true},
		{name: "invalid base", base: "GBP", want: domain.ErrInvalidCurrencyType},
		{name: "non USD base", base: "EUR", rates: map[string]float32{"EUR": 1}, invalid: true},
		{name: "empty rates", base: "USD", invalid: true},
		{name: "missing base", base: "USD", rates: map[string]float32{"EUR": 0.85}, invalid: true},
		{name: "invalid currency", base: "USD", rates: map[string]float32{"USD": 1, "GBP": 1}, want: domain.ErrInvalidCurrencyType},
		{name: "wrong base rate", base: "USD", rates: map[string]float32{"USD": 2}, want: domain.ErrInvalidBaseCurrencyRate},
		{name: "NaN rate", base: "USD", rates: map[string]float32{"USD": 1, "EUR": float32(math.NaN())}, want: domain.ErrInvalidExchangeRateValue},
		{name: "negative rate", base: "USD", rates: map[string]float32{"USD": 1, "EUR": -1}, want: domain.ErrInvalidExchangeRateValue},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			response := &exchange.ExchangeRatesResponse{BaseCurrency: tt.base, Rates: tt.rates}
			calls := 0
			adapter := client.New(exchangeStub{get: func(got context.Context, req *exchange.Empty) (*exchange.ExchangeRatesResponse, error) {
				calls++
				if got != ctx {
					t.Error("caller context not forwarded")
				}
				if req == nil {
					t.Error("nil RPC request")
				}
				if tt.rpcErr != nil {
					return nil, tt.rpcErr
				}
				return response, nil
			}})
			got, err := adapter.GetExchangeRates(ctx)
			if calls != 1 {
				t.Errorf("RPC calls=%d; want 1", calls)
			}
			if tt.want != nil || tt.invalid {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.want != nil && !errors.Is(err, tt.want) {
					t.Errorf("error=%v; want %v", err, tt.want)
				}
				if tt.rpcErr != nil && status.Code(err) != status.Code(tt.rpcErr) {
					t.Error("RPC status lost")
				}
				if !reflect.DeepEqual(got, domain.ExchangeRates{}) {
					t.Error("expected zero result on failure")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := domain.Rates{"USD": 1, "EUR": 0.85, "RUB": 90}
			if got.BaseCurrency().CurrencyType() != domain.CurrencyTypeUSD || !reflect.DeepEqual(got.Rates(), want) {
				t.Fatal("incorrect response mapping")
			}
			response.Rates["EUR"] = 0
			if !reflect.DeepEqual(got.Rates(), want) {
				t.Error("RPC response mutation changed domain model")
			}
		})
	}
}
