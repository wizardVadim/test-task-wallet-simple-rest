package auth_http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	authhttp "wallet-app/internal/features/auth/transport/http"
)

type validatorStub func(string) (string, error)

func (v validatorStub) Validate(value string) (string, error) { return v(value) }

func TestAuthenticateSuccess(t *testing.T) {
	userID := uuid.MustParse("e373039e-7060-4525-90bc-d5c7d7525a80")
	for _, header := range []string{"Bearer signed-token", "bearer signed-token", "  bEaReR\t signed-token  "} {
		t.Run(header, func(t *testing.T) {
			type originalKey struct{}
			ctx := context.WithValue(t.Context(), originalKey{}, "preserved")
			request := httptest.NewRequest(http.MethodPost, "/private", strings.NewReader(`{"amount":100}`)).WithContext(ctx)
			request.Header.Set("Authorization", header)
			validateCalls, nextCalls := 0, 0
			middleware := authhttp.Authenticate(validatorStub(func(value string) (string, error) {
				validateCalls++
				if value != "signed-token" {
					t.Errorf("token = %q", value)
				}
				return userID.String(), nil
			}), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalls++
				got, ok := authhttp.UserIDFromContext(r.Context())
				if !ok || got != userID {
					t.Error("missing or incorrect user ID")
				}
				if r.Context().Value(originalKey{}) != "preserved" {
					t.Error("original context lost")
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != `{"amount":100}` {
					t.Error("request body modified")
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			response := httptest.NewRecorder()
			middleware.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent || validateCalls != 1 || nextCalls != 1 {
				t.Errorf("status/calls = %d/%d/%d", response.Code, validateCalls, nextCalls)
			}
			if _, ok := authhttp.UserIDFromContext(request.Context()); ok {
				t.Error("original request context modified")
			}
		})
	}
}

func TestAuthenticateRejectsInvalidRequests(t *testing.T) {
	for _, tt := range []struct {
		name, header, subject string
		err                   error
		wantValidate          int
	}{
		{name: "missing header"},
		{name: "empty bearer", header: "Bearer"},
		{name: "wrong scheme", header: "Basic value"},
		{name: "extra field", header: "Bearer token extra"},
		{name: "invalid token", header: "Bearer token", err: errors.New("invalid signature"), wantValidate: 1},
		{name: "empty subject", header: "Bearer token", wantValidate: 1},
		{name: "malformed UUID", header: "Bearer token", subject: "user-42", wantValidate: 1},
		{name: "nil UUID", header: "Bearer token", subject: uuid.Nil.String(), wantValidate: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls, nextCalls := 0, 0
			middleware := authhttp.Authenticate(validatorStub(func(string) (string, error) { calls++; return tt.subject, tt.err }), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalls++ }))
			request := httptest.NewRequest(http.MethodGet, "/private", nil)
			request.Header.Set("Authorization", tt.header)
			response := httptest.NewRecorder()
			middleware.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Errorf("status = %d; want 401", response.Code)
			}
			if calls != tt.wantValidate {
				t.Errorf("Validate calls = %d; want %d", calls, tt.wantValidate)
			}
			if nextCalls != 0 {
				t.Error("unauthorized request reached handler")
			}
		})
	}
}

func TestAuthenticateCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	middleware := authhttp.Authenticate(validatorStub(func(string) (string, error) { t.Fatal("validator called after cancellation"); return "", nil }), http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler called after cancellation") }))
	request := httptest.NewRequest(http.MethodGet, "/private", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	middleware.ServeHTTP(response, request)
	if response.Body.Len() != 0 || len(response.Header()) != 0 {
		t.Error("response written after cancellation")
	}
}

func TestUserIDFromEmptyContext(t *testing.T) {
	id, ok := authhttp.UserIDFromContext(context.Background())
	if ok || id != uuid.Nil {
		t.Errorf("ID = %v, found = %v", id, ok)
	}
}
