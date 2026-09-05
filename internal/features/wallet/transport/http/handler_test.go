package wallet_http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"wallet-app/internal/features/wallet/service"

	"github.com/google/uuid"

	"wallet-app/internal/core/domain"
)

type balanceServiceStub struct {
	WalletService
	get func(context.Context, domain.WalletID) (int64, error)
}

func (s balanceServiceStub) GetWalletBalance(ctx context.Context, id domain.WalletID) (int64, error) {
	return s.get(ctx, id)
}

func TestGetWalletBalance(t *testing.T) {
	const validID = "550e8400-e29b-41d4-a716-446655440000"
	for _, tt := range []struct {
		name, id      string
		balance       int64
		err           error
		status, calls int
		message       ErrorMessage
	}{
		{name: "success", id: validID, balance: 100, status: 200, calls: 1},
		{name: "zero balance", id: validID, status: 200, calls: 1},
		{name: "invalid UUID", id: "invalid", status: 400, message: ErrorInvalidWalletID},
		{name: "nil UUID", id: "00000000-0000-0000-0000-000000000000", status: 400, message: ErrorInvalidWalletID},
		{name: "not found", id: validID, err: fmt.Errorf("read: %w", domain.ErrWalletNotFound), status: 404, calls: 1, message: ErrorWalletNotFound},
		{name: "internal error", id: validID, err: errors.New("private database details"), status: 500, calls: 1, message: ErrorGetBalanceInternal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+tt.id, nil)
			calls := 0
			svc := balanceServiceStub{get: func(ctx context.Context, id domain.WalletID) (int64, error) {
				calls++
				if ctx != request.Context() || id.Value().String() != tt.id {
					t.Fatal("incorrect service arguments")
				}
				return tt.balance, tt.err
			}}
			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/v1/wallets/{wallet_uuid}", New(svc).GetWalletBalance)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if calls != tt.calls {
				t.Fatalf("service calls = %d; want %d", calls, tt.calls)
			}
			if response.Code != tt.status {
				t.Fatalf("status = %d; want %d", response.Code, tt.status)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q", got)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tt.message != "" {
				var got ErrorDTO
				if err := json.Unmarshal(body["error"], &got); err != nil {
					t.Fatal(err)
				}
				if got.Message != tt.message {
					t.Fatalf("message = %q; want %q", got.Message, tt.message)
				}
				if string(body["payload"]) != "null" {
					t.Fatalf("unexpected payload: %s", body["payload"])
				}
			} else {
				if _, ok := body["error"]; ok {
					t.Fatal("unexpected error field")
				}
				var payload GetBalancePayload
				if err := json.Unmarshal(body["payload"], &payload); err != nil {
					t.Fatal(err)
				}
				if payload.Balance != tt.balance {
					t.Fatalf("balance = %d; want %d", payload.Balance, tt.balance)
				}
			}
		})
	}
}

func TestGetWalletBalanceCancelledContext(t *testing.T) {
	for _, duringCall := range []bool{false, true} {
		t.Run(fmt.Sprintf("during_service_call=%t", duringCall), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if !duringCall {
				cancel()
			}
			calls := 0
			svc := balanceServiceStub{get: func(context.Context, domain.WalletID) (int64, error) {
				calls++
				cancel()
				return 0, context.Canceled
			}}
			request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			request.SetPathValue("wallet_uuid", "550e8400-e29b-41d4-a716-446655440000")
			response := httptest.NewRecorder()
			New(svc).GetWalletBalance(response, request)
			wantCalls := 0
			if duringCall {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("service calls = %d; want %d", calls, wantCalls)
			}
			if response.Body.Len() != 0 {
				t.Fatalf("unexpected response: %s", response.Body.String())
			}
		})
	}
}

type mutationServiceStub struct {
	WalletService
	create func(context.Context) (domain.Wallet, error)
	change func(context.Context, domain.WalletOperation) error
}

func (s mutationServiceStub) CreateNewWallet(ctx context.Context) (domain.Wallet, error) {
	return s.create(ctx)
}
func (s mutationServiceStub) ChangeWalletBalance(ctx context.Context, op domain.WalletOperation) error {
	return s.change(ctx, op)
}

func checkJSONResponse(t *testing.T, response *httptest.ResponseRecorder, status int, message ErrorMessage) map[string]json.RawMessage {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d; want %d; body = %s", response.Code, status, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if message == "" {
		if _, exists := body["error"]; exists {
			t.Fatalf("unexpected error: %s", body["error"])
		}
	} else {
		var got ErrorDTO
		if err := json.Unmarshal(body["error"], &got); err != nil {
			t.Fatal(err)
		}
		if got.Message != message {
			t.Fatalf("message = %q; want %q", got.Message, message)
		}
		if string(body["payload"]) != "null" {
			t.Fatalf("unexpected payload: %s", body["payload"])
		}
	}
	return body
}

func TestCreateWallet(t *testing.T) {
	id, err := domain.NewWalletID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := domain.NewWallet(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		err     error
		status  int
		message ErrorMessage
	}{
		{name: "created", status: http.StatusCreated},
		{name: "service error", err: errors.New("private database details"), status: http.StatusInternalServerError, message: ErrorCreateWalletInternal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/wallets", nil)
			calls := 0
			stub := mutationServiceStub{create: func(ctx context.Context) (domain.Wallet, error) {
				calls++
				if ctx != request.Context() {
					t.Fatal("incorrect context")
				}
				if tt.err != nil {
					return domain.Wallet{}, tt.err
				}
				return wallet, nil
			}}
			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/wallets", New(stub).CreateWallet)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if calls != 1 {
				t.Fatalf("service calls = %d; want 1", calls)
			}
			body := checkJSONResponse(t, response, tt.status, tt.message)
			if tt.err != nil {
				if response.Header().Get("Location") != "" {
					t.Fatal("unexpected Location on error")
				}
				return
			}
			if got := response.Header().Get("Location"); got != "/api/v1/wallets/"+id.Value().String() {
				t.Fatalf("Location = %q", got)
			}
			var payload struct {
				WalletID string `json:"walletId"`
				Balance  *int64 `json:"balance"`
			}
			if err := json.Unmarshal(body["payload"], &payload); err != nil {
				t.Fatal(err)
			}
			if payload.WalletID != id.Value().String() || payload.Balance == nil || *payload.Balance != 0 {
				t.Fatalf("unexpected payload: %s", body["payload"])
			}
		})
	}
}

func TestChangeWalletBalance(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	const valid = `{"walletId":"` + id + `","operationType":"DEPOSIT","amount":100}`
	tests := []struct {
		name, body    string
		err           error
		status, calls int
		message       ErrorMessage
		operation     domain.OperationType
	}{
		{name: "deposit", body: valid, status: 200, calls: 1, operation: domain.OperationTypeDeposit},
		{name: "withdraw", body: strings.Replace(valid, "DEPOSIT", "WITHDRAW", 1), status: 200, calls: 1, operation: domain.OperationTypeWithdraw},
		{name: "trailing whitespace", body: valid + " \n\t", status: 200, calls: 1, operation: domain.OperationTypeDeposit},
		{name: "exact body limit", body: valid + strings.Repeat(" ", 4096-len(valid)), status: 200, calls: 1, operation: domain.OperationTypeDeposit},
		{name: "empty body", status: 400, message: ErrorInvalidRequestBody},
		{name: "malformed JSON", body: `{`, status: 400, message: ErrorInvalidRequestBody},
		{name: "null", body: `null`, status: 400, message: ErrorInvalidRequestBody},
		{name: "array", body: `[]`, status: 400, message: ErrorInvalidRequestBody},
		{name: "missing fields", body: `{}`, status: 400, message: ErrorInvalidRequestBody},
		{name: "invalid UUID", body: strings.Replace(valid, id, "invalid", 1), status: 400, message: ErrorInvalidRequestBody},
		{name: "nil UUID", body: strings.Replace(valid, id, "00000000-0000-0000-0000-000000000000", 1), status: 400, message: ErrorInvalidWalletID},
		{name: "invalid operation", body: strings.Replace(valid, "DEPOSIT", "INVALID", 1), status: 400, message: ErrorInvalidRequestBody},
		{name: "zero amount", body: strings.Replace(valid, "100", "0", 1), status: 400, message: ErrorInvalidRequestBody},
		{name: "negative amount", body: strings.Replace(valid, "100", "-1", 1), status: 400, message: ErrorInvalidRequestBody},
		{name: "fractional amount", body: strings.Replace(valid, "100", "1.5", 1), status: 400, message: ErrorInvalidRequestBody},
		{name: "string amount", body: strings.Replace(valid, "100", `"100"`, 1), status: 400, message: ErrorInvalidRequestBody},
		{name: "amount exceeds int64", body: strings.Replace(valid, "100", "9223372036854775808", 1), status: 400, message: ErrorInvalidRequestBody},
		{name: "second JSON object", body: valid + `{}`, status: 400, message: ErrorInvalidRequestBody},
		{name: "trailing junk", body: valid + `junk`, status: 400, message: ErrorInvalidRequestBody},
		{name: "oversized first value", body: `{"walletId":"` + strings.Repeat("a", 4096) + `"}`, status: 413, message: ErrorRequestBodyTooLarge},
		{name: "oversized trailing whitespace", body: valid + strings.Repeat(" ", 4097-len(valid)), status: 413, message: ErrorRequestBodyTooLarge},
		{name: "not found", body: valid, err: fmt.Errorf("change: %w", domain.ErrWalletNotFound), status: 404, calls: 1, message: ErrorWalletNotFound},
		{name: "insufficient funds", body: valid, err: fmt.Errorf("change: %w", service.ErrSmallBalance), status: 409, calls: 1, message: ErrorInsufficientFunds},
		{name: "balance overflow", body: valid, err: fmt.Errorf("change: %w", service.ErrBalanceOverflow), status: 422, calls: 1, message: ErrorBalanceOverflow},
		{name: "internal error", body: valid, err: errors.New("private database details"), status: 500, calls: 1, message: ErrorChangeBalanceInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", "application/json")
			calls := 0
			stub := mutationServiceStub{change: func(ctx context.Context, op domain.WalletOperation) error {
				calls++
				wantType := tt.operation
				if wantType == "" {
					wantType = domain.OperationTypeDeposit
				}
				if ctx != request.Context() || op.WalletID().Value().String() != id || op.Amount() != 100 || op.OperationType() != wantType {
					t.Fatalf("incorrect service arguments: %+v", op)
				}
				return tt.err
			}}
			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/v1/wallet", New(stub).ChangeWalletBalance)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if calls != tt.calls {
				t.Fatalf("service calls = %d; want %d", calls, tt.calls)
			}
			body := checkJSONResponse(t, response, tt.status, tt.message)
			if tt.message == "" && string(body["payload"]) != "null" {
				t.Fatalf("unexpected payload: %s", body["payload"])
			}
		})
	}
}

func TestMutationHandlersCancelledContext(t *testing.T) {
	for _, name := range []string{"create", "change"} {
		for _, duringCall := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/during_service_call=%t", name, duringCall), func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if !duringCall {
					cancel()
				}
				calls := 0
				stub := mutationServiceStub{
					create: func(context.Context) (domain.Wallet, error) {
						calls++
						cancel()
						return domain.Wallet{}, context.Canceled
					},
					change: func(context.Context, domain.WalletOperation) error { calls++; cancel(); return context.Canceled },
				}
				request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"walletId":"550e8400-e29b-41d4-a716-446655440000","operationType":"DEPOSIT","amount":100}`)).WithContext(ctx)
				response := httptest.NewRecorder()
				handler := New(stub)
				if name == "create" {
					handler.CreateWallet(response, request)
				} else {
					handler.ChangeWalletBalance(response, request)
				}
				wantCalls := 0
				if duringCall {
					wantCalls = 1
				}
				if calls != wantCalls {
					t.Fatalf("service calls = %d; want %d", calls, wantCalls)
				}
				if response.Body.Len() != 0 {
					t.Fatalf("unexpected response: %s", response.Body.String())
				}
			})
		}
	}
}
