package domain_test

import (
	"errors"
	"testing"

	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	validID := uuid.New()
	for _, tt := range []struct {
		name     string
		id       uuid.UUID
		username string
		email    string
		hash     string
		wantErr  error
	}{
		{"valid", validID, "ivan", "ivan@example.com", "test-hash", nil},
		{"nil ID", uuid.Nil, "ivan", "ivan@example.com", "test-hash", domain.ErrInvalidUserID},
		{"empty username", validID, "", "ivan@example.com", "test-hash", domain.ErrInvalidUsername},
		{"whitespace username", validID, " \t\n ", "ivan@example.com", "test-hash", domain.ErrInvalidUsername},
		{"malformed email", validID, "ivan", "not-an-address", "test-hash", domain.ErrInvalidEmailAddress},
		{"empty email", validID, "ivan", "", "test-hash", domain.ErrInvalidEmailAddress},
		{"empty hash", validID, "ivan", "ivan@example.com", "", domain.ErrInvalidPasswordHash},
	} {
		t.Run(tt.name, func(t *testing.T) {
			user, err := domain.NewUser(tt.id, tt.username, tt.email, tt.hash)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewUser() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && user != (domain.User{}) {
				t.Error("constructor returned a populated user on error")
			}
			if tt.wantErr == nil && user == (domain.User{}) {
				t.Error("constructor returned an empty user on success")
			}
		})
	}
}

func TestUserPreservesFieldsAndNormalizesUsername(t *testing.T) {
	id := uuid.New()
	const email = "ivan@example.com"
	const hash = "$test$CaseSensitiveHash"
	user, err := domain.NewUser(id, " \tIvAn \n", email, hash)
	if err != nil {
		t.Fatal(err)
	}
	if got := user.ID().String(); got != id.String() {
		t.Errorf("ID = %q, want %q", got, id)
	}
	if got := user.Username(); got != "ivan" {
		t.Errorf("Username = %q, want ivan", got)
	}
	if got := user.Email(); got != email {
		t.Errorf("Email = %q, want %q", got, email)
	}
	if user.PasswordHash() != hash {
		t.Error("PasswordHash must preserve the original hash")
	}
}
