package grpc

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"contracts/exchange"
	"exchanger-app/internal/core/domain"
	"exchanger-app/internal/features/rates/service"
	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestTransportWire(t *testing.T) {
	usd := mustCurrency(t, domain.CurrencyUSD)
	eur := mustCurrency(t, domain.CurrencyEUR)
	rates := []domain.ExchangeRate{mustExchangeRate(t, 1, usd, usd), mustExchangeRate(t, 0.87, usd, eur)}
	handler := New(&stubRateService{
		getAllFn: func(context.Context) ([]domain.ExchangeRate, error) { return rates, nil },
		getRateFn: func(_ context.Context, from, to domain.Currency) (domain.ExchangeRate, error) {
			if !from.IsEqual(usd) || !to.IsEqual(eur) {
				return domain.ExchangeRate{}, service.ErrNotFoundCurrency
			}
			return rates[1], nil
		},
	})
	listener := bufconn.Listen(1024 * 1024)
	server := grpcgo.NewServer()
	exchange.RegisterExchangeServiceServer(server, handler)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		listener.Close()
		if err := <-done; err != nil && !errors.Is(err, grpcgo.ErrServerStopped) {
			t.Errorf("Serve: %v", err)
		}
	})
	conn, err := grpcgo.NewClient("passthrough:///test", grpcgo.WithTransportCredentials(insecure.NewCredentials()), grpcgo.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	client := exchange.NewExchangeServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	all, err := client.GetExchangeRates(ctx, &exchange.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if all.GetBaseCurrency() != "USD" || len(all.GetRates()) != 2 || all.GetRates()["USD"] != 1 || all.GetRates()["EUR"] != float32(0.87) {
		t.Fatalf("unexpected response: %v", all)
	}
	pair, err := client.GetExchangeRateForCurrency(ctx, &exchange.CurrencyRequest{FromCurrency: "usd", ToCurrency: "eur"})
	if err != nil {
		t.Fatal(err)
	}
	if pair.GetFromCurrency() != "USD" || pair.GetToCurrency() != "EUR" || pair.GetRate() != float32(0.87) {
		t.Fatalf("unexpected pair: %v", pair)
	}
	for _, tt := range []struct {
		from, to string
		code     codes.Code
	}{{"GBP", "USD", codes.InvalidArgument}, {"USD", "RUB", codes.NotFound}} {
		_, err := client.GetExchangeRateForCurrency(ctx, &exchange.CurrencyRequest{FromCurrency: tt.from, ToCurrency: tt.to})
		if status.Code(err) != tt.code {
			t.Errorf("%s/%s: got %v, want %v", tt.from, tt.to, err, tt.code)
		}
	}
}

func TestMapErrorWrapped(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		code codes.Code
	}{
		{"canceled", context.Canceled, codes.Canceled},
		{"deadline", context.DeadlineExceeded, codes.DeadlineExceeded},
		{"invalid stored currency", domain.ErrInvalidCurrencyType, codes.Internal},
		{"missing", service.ErrNotFoundCurrency, codes.NotFound},
		{"internal", errors.New("private database details"), codes.Internal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := mapError(errors.Join(errors.New("outer error"), tt.err))
			if status.Code(got) != tt.code {
				t.Fatalf("got %v, want %v", got, tt.code)
			}
			if tt.code == codes.Internal && status.Convert(got).Message() != "internal server error" {
				t.Fatalf("internal error details leaked: %v", got)
			}
		})
	}
}
