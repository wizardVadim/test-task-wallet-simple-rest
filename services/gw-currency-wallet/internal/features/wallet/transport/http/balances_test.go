package wallet_http

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"wallet-app/internal/core/domain"
	authhttp "wallet-app/internal/features/auth/transport/http"
)

type balancesServiceStub struct {
	WalletService
	getBalances func(context.Context, uuid.UUID) ([]domain.Balance, error)
}

func (s balancesServiceStub) GetBalances(ctx context.Context, id uuid.UUID) ([]domain.Balance, error) {
	return s.getBalances(ctx, id)
}

type tokenValidatorStub func(string) (string, error)

func (v tokenValidatorStub) Validate(value string) (string, error) { return v(value) }

func TestGetBalancesResponse(t *testing.T) {
	id := uuid.New()
	for _, tt := range []struct {
		name    string
		amounts map[string]int64
		want    map[string]string
	}{
		{"decimal precision", map[string]int64{"USD": 12345, "EUR": 5, "RUB": math.MaxInt64}, map[string]string{"USD": "123.45", "EUR": "0.05", "RUB": "92233720368547758.07"}},
		{"zero", map[string]int64{"USD": 0}, map[string]string{"USD": "0.00"}},
		{"empty", map[string]int64{}, map[string]string{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			balances := []domain.Balance{}
			for code, amount := range tt.amounts {
				currency, err := domain.NewCurrency(domain.CurrencyType(code))
				if err != nil {
					t.Fatal(err)
				}
				balance, err := domain.NewBalance(id, currency, amount)
				if err != nil {
					t.Fatal(err)
				}
				balances = append(balances, balance)
			}
			calls := 0
			svc := balancesServiceStub{getBalances: func(ctx context.Context, gotID uuid.UUID) ([]domain.Balance, error) {
				calls++
				contextID, ok := authhttp.UserIDFromContext(ctx)
				if !ok || gotID != id || contextID != id {
					t.Error("incorrect user ID")
				}
				return balances, nil
			}}
			handler := authhttp.Authenticate(tokenValidatorStub(func(string) (string, error) { return id.String(), nil }), http.HandlerFunc(New(svc).GetBalances))
			request := httptest.NewRequest(http.MethodGet, "/api/v1/balance", nil)
			request.Header.Set("Authorization", "Bearer token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 200 || calls != 1 {
				t.Fatalf("status/calls = %d/%d", response.Code, calls)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Error("missing JSON content type")
			}
			var body struct {
				Balance map[string]json.RawMessage `json:"balance"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Balance == nil || len(body.Balance) != len(tt.want) {
				t.Fatalf("incorrect balance map: %s", response.Body.String())
			}
			for code, want := range tt.want {
				if string(body.Balance[code]) != want {
					t.Errorf("%s = %s; want unquoted %s", code, body.Balance[code], want)
				}
			}
		})
	}
}

func TestGetBalancesHandlerFailures(t *testing.T) {
	for _, authenticated := range []bool{false, true} {
		name := "missing user context"
		if authenticated {
			name = "service error"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			svc := balancesServiceStub{getBalances: func(context.Context, uuid.UUID) ([]domain.Balance, error) {
				calls++
				return nil, errors.New("private database failure")
			}}
			var handler http.Handler = http.HandlerFunc(New(svc).GetBalances)
			if authenticated {
				handler = authhttp.Authenticate(tokenValidatorStub(func(string) (string, error) { return uuid.NewString(), nil }), handler)
			}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/balance", nil)
			request.Header.Set("Authorization", "Bearer token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			wantCalls := 0
			if authenticated {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Errorf("service calls = %d; want %d", calls, wantCalls)
			}
			// An absent context ID behind the middleware is an internal wiring failure.
			if response.Code != 500 {
				t.Errorf("status = %d; want 500", response.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("response must be a single JSON value: %v; body=%s", err, response.Body.String())
			}
			if len(body) != 1 || body["error"] != "internal server error" {
				t.Errorf("unexpected body: %v", body)
			}
		})
	}
}

func TestGetBalancesHandlerCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := balancesServiceStub{getBalances: func(context.Context, uuid.UUID) ([]domain.Balance, error) {
		t.Fatal("service called after cancellation")
		return nil, nil
	}}
	response := httptest.NewRecorder()
	New(svc).GetBalances(response, httptest.NewRequest(http.MethodGet, "/api/v1/balance", nil).WithContext(ctx))
	if response.Body.Len() != 0 {
		t.Error("response written after cancellation")
	}
}
