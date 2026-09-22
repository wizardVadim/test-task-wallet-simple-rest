package auth_http_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"wallet-app/internal/core/domain"
	authhttp "wallet-app/internal/features/auth/transport/http"

	passwordhash "wallet-app/internal/core/infrastructure/hash"
	"wallet-app/internal/features/auth/service"

	"github.com/google/uuid"
)

type authStub struct {
	register func(context.Context, string, string, string) (domain.User, error)
	login    func(context.Context, string, string) (domain.User, string, error)
}

func (s authStub) Register(ctx context.Context, username, email, password string) (domain.User, error) {
	return s.register(ctx, username, email, password)
}

func (s authStub) Login(ctx context.Context, username, password string) (domain.User, string, error) {
	return s.login(ctx, username, password)
}

func TestRegisterSuccess(t *testing.T) {
	user, err := domain.NewUser(uuid.New(), "ivan", "ivan@example.com", "secret-hash")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader(`{"username":"ivan","email":"ivan@example.com","password":" secret "}`))
	calls := 0
	handler := authhttp.New(authStub{register: func(ctx context.Context, username, email, password string) (domain.User, error) {
		calls++
		if ctx != request.Context() || username != "ivan" || email != "ivan@example.com" || password != " secret " {
			t.Error("incorrect service arguments")
		}
		return user, nil
	}})
	response := httptest.NewRecorder()
	handler.Register(response, request)
	if calls != 1 {
		t.Errorf("service calls = %d; want 1", calls)
	}
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201", response.Code)
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Error("missing JSON content type")
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["message"] != "User registered successfully" {
		t.Errorf("unexpected success body: %v", body)
	}
}

func TestRegisterInvalidJSON(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
	}{
		{"empty", "", 400},
		{"malformed", "{", 400},
		{"wrong field type", `{"username":123}`, 400},
		{"array", `[]`, 400},
		{"multiple values", `{} {}`, 400},
		{"trailing garbage", `{} x`, 400},
		{"oversized", `{"username":"` + strings.Repeat("x", 4096) + `"}`, 413},
		{"oversized trailing whitespace", `{}` + strings.Repeat(" ", 4096), 413},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handler := authhttp.New(authStub{register: func(context.Context, string, string, string) (domain.User, error) {
				t.Fatal("service called for invalid JSON")
				return domain.User{}, nil
			}})
			response := httptest.NewRecorder()
			handler.Register(response, httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader(tt.body)))
			if response.Code != tt.status {
				t.Errorf("status = %d; want %d", response.Code, tt.status)
			}
		})
	}
}

func TestRegisterServiceErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"invalid username", domain.ErrInvalidUsername, 400},
		{"invalid email", domain.ErrInvalidEmailAddress, 400},
		{"invalid password", domain.ErrInvalidPassword, 400},
		{"duplicate username", domain.ErrUsernameAlreadyExists, 400},
		{"duplicate email", domain.ErrEmailAlreadyExists, 400},
		{"internal", errors.New("private database error"), 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handler := authhttp.New(authStub{register: func(context.Context, string, string, string) (domain.User, error) {
				return domain.User{}, fmt.Errorf("register: %w", tt.err)
			}})
			response := httptest.NewRecorder()
			handler.Register(response, httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader(`{"username":"ivan","email":"ivan@example.com","password":"secret"}`)))
			if response.Code != tt.status {
				t.Errorf("status = %d; want %d", response.Code, tt.status)
			}
			if strings.Contains(response.Body.String(), "private database error") {
				t.Error("internal details leaked in response")
			}
			if tt.name == "duplicate username" || tt.name == "duplicate email" {
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if len(body) != 1 || body["error"] != "Username or email already exists" {
					t.Errorf("duplicate response does not match API contract: %v", body)
				}
			}
		})
	}
}

func TestRegisterCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	handler := authhttp.New(authStub{register: func(context.Context, string, string, string) (domain.User, error) {
		t.Fatal("service called with canceled context")
		return domain.User{}, nil
	}})
	response := httptest.NewRecorder()
	handler.Register(response, httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader(`{}`)).WithContext(ctx))
	if response.Body.Len() != 0 {
		t.Error("response written after cancellation")
	}
}

// Exercise the real service validation instead of exposing a bcrypt error to HTTP.
func TestRegisterPasswordTooLong(t *testing.T) {
	handler := authhttp.New(service.New(nil, nil, &passwordhash.BcryptHasher{}, nil))
	body := `{"username":"ivan","email":"ivan@example.com","password":"` + strings.Repeat("я", 36) + `a"}`
	response := httptest.NewRecorder()
	handler.Register(response, httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader(body)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400", response.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["error"] != "invalid user password" {
		t.Errorf("unexpected response: %v", got)
	}
}
