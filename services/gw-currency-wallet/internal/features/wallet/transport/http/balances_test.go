package wallet_http

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
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

type operationServiceStub struct {
	WalletService
	apply func(context.Context, domain.BalanceOperation) ([]domain.Balance, error)
}

func (s operationServiceStub) ApplyBalanceOperation(ctx context.Context, op domain.BalanceOperation) ([]domain.Balance, error) {
	return s.apply(ctx, op)
}

func TestAmountConversion(t *testing.T) {
	for _, tt := range []struct {
		input string
		want  int64
	}{
		{"0", 0}, {"1", 100}, {"1.2", 120}, {"1.23", 123}, {"0.01", 1}, {"0.29", 29}, {"90071992547409.93", 9007199254740993}, {"92233720368547758.07", math.MaxInt64},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := jsonNumberToAmountInt64(json.Number(tt.input))
			if err != nil || got != tt.want {
				t.Errorf("got %d,%v; want %d", got, err, tt.want)
			}
		})
	}
	for _, input := range []string{"", "1e2", "1E2", "1.234", "92233720368547758.08", "92233720368547759", "1.2.3", "abc"} {
		t.Run("reject_"+input, func(t *testing.T) {
			if _, err := jsonNumberToAmountInt64(json.Number(input)); !errors.Is(err, errInvalidInputAmount) {
				t.Errorf("error=%v; want invalid amount", err)
			}
		})
	}
}

func TestBalanceDepositWithdraw(t *testing.T) {
	id := uuid.New()
	for _, withdraw := range []bool{false, true} {
		name := "deposit"
		kind := domain.OperationTypeDeposit
		message := "Account topped up successfully"
		if withdraw {
			name = "withdraw"
			kind = domain.OperationTypeWithdraw
			message = "Withdrawal successful"
		}
		t.Run(name, func(t *testing.T) {
			currency, err := domain.NewCurrency(domain.CurrencyTypeUSD)
			if err != nil {
				t.Fatal(err)
			}
			balance, err := domain.NewBalance(id, currency, 12345)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			svc := operationServiceStub{apply: func(ctx context.Context, op domain.BalanceOperation) ([]domain.Balance, error) {
				calls++
				contextID, ok := authhttp.UserIDFromContext(ctx)
				if !ok || contextID != id || op.UserID() != id || op.Currency() != currency || op.Amount() != 29 || op.OperationType() != kind {
					t.Error("incorrect operation arguments")
				}
				return []domain.Balance{balance}, nil
			}}
			h := New(svc)
			endpoint := h.BalanceDeposit
			if withdraw {
				endpoint = h.BalanceWithdraw
			}
			handler := authhttp.Authenticate(tokenValidatorStub(func(string) (string, error) { return id.String(), nil }), http.HandlerFunc(endpoint))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/"+name, strings.NewReader(`{"currency":"USD","amount":0.29,"user_id":"ignored-attacker-id"}`))
			request.Header.Set("Authorization", "Bearer token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != 200 || calls != 1 {
				t.Fatalf("status/calls=%d/%d", response.Code, calls)
			}
			var body struct {
				Message  string                     `json:"message"`
				Balances map[string]json.RawMessage `json:"new_balance"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Message != message || string(body.Balances["USD"]) != "123.45" {
				t.Errorf("incorrect response: %s", response.Body.String())
			}
		})
	}
}

func TestBalanceOperationFailures(t *testing.T) {
	for _, tt := range []struct {
		name, body    string
		serviceErr    error
		status, calls int
	}{
		{name: "invalid JSON", body: "{", status: 400}, {name: "multiple values", body: "{} {}", status: 400},
		{name: "oversized", body: strings.Repeat(" ", 4097), status: 413},
		{name: "negative", body: `{"currency":"USD","amount":-0.01}`, status: 400},
		{name: "zero", body: `{"currency":"USD","amount":0}`, status: 400},
		{name: "missing amount", body: `{"currency":"USD"}`, status: 400},
		{name: "null amount", body: `{"currency":"USD","amount":null}`, status: 400},
		{name: "precision", body: `{"currency":"USD","amount":0.001}`, status: 400},
		{name: "overflow input", body: `{"currency":"USD","amount":92233720368547758.08}`, status: 400},
		{name: "exponent", body: `{"currency":"USD","amount":1e2}`, status: 400},
		{name: "currency", body: `{"currency":"GBP","amount":1}`, status: 400},
		{name: "insufficient funds", body: `{"currency":"USD","amount":1}`, serviceErr: domain.ErrSmallBalance, status: 400, calls: 1},
		{name: "balance overflow", body: `{"currency":"USD","amount":1}`, serviceErr: domain.ErrBalanceOverflow, status: 422, calls: 1},
		{name: "database error", body: `{"currency":"USD","amount":1}`, serviceErr: errors.New("private database failure"), status: 500, calls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			svc := operationServiceStub{apply: func(context.Context, domain.BalanceOperation) ([]domain.Balance, error) {
				calls++
				return nil, tt.serviceErr
			}}
			handler := authhttp.Authenticate(tokenValidatorStub(func(string) (string, error) { return uuid.NewString(), nil }), http.HandlerFunc(New(svc).BalanceWithdraw))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/withdraw", strings.NewReader(tt.body))
			request.Header.Set("Authorization", "Bearer token")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tt.status || calls != tt.calls {
				t.Errorf("status/calls=%d/%d; want %d/%d", response.Code, calls, tt.status, tt.calls)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 1 || body["error"] == nil || strings.Contains(response.Body.String(), "private database failure") {
				t.Errorf("incorrect error body: %s", response.Body.String())
			}
		})
	}
}

func TestBalanceOperationMissingAuthenticationAndCancellation(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		name := "missing context"
		if canceled {
			name = "canceled context"
		}
		t.Run(name, func(t *testing.T) {
			svc := operationServiceStub{apply: func(context.Context, domain.BalanceOperation) ([]domain.Balance, error) {
				t.Fatal("service must not be called")
				return nil, nil
			}}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if canceled {
				cancel()
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/deposit", strings.NewReader(`{"amount":1,"currency":"USD"}`)).WithContext(ctx)
			response := httptest.NewRecorder()
			New(svc).BalanceDeposit(response, request)
			if canceled {
				if response.Body.Len() != 0 {
					t.Error("response written after cancellation")
				}
			} else if response.Code != 500 {
				t.Errorf("missing context status=%d; want 500", response.Code)
			}
		})
	}
}
