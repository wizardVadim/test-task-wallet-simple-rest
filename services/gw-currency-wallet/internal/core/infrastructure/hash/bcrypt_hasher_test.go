package hash_test

import (
	"errors"
	"strings"
	"testing"

	"wallet-app/internal/core/domain"
	passwordhash "wallet-app/internal/core/infrastructure/hash"
	"wallet-app/internal/features/auth/service"

	"golang.org/x/crypto/bcrypt"
)

var _ service.PasswordHasher = (*passwordhash.BcryptHasher)(nil)

func TestBcryptHashAndCompare(t *testing.T) {
	hasher := &passwordhash.BcryptHasher{}
	password := " secret пароль "
	first, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	if first == password || second == password {
		t.Fatal("stored plaintext instead of a hash")
	}
	if first == second {
		t.Error("repeated hashing must use independent salts")
	}
	for _, value := range []string{first, second} {
		if err := hasher.Compare(password, value); err != nil {
			t.Errorf("correct password rejected: %v", err)
		}
	}
	for _, wrong := range []string{"wrong password", "", strings.TrimSpace(password)} {
		if err := hasher.Compare(wrong, first); !errors.Is(err, domain.ErrPasswordMismatch) {
			t.Errorf("incorrect password: error = %v; want mismatch", err)
		}
	}
}

func TestBcryptPasswordLength(t *testing.T) {
	hasher := &passwordhash.BcryptHasher{}
	if hasher.MaxByteLength() != 72 {
		t.Fatalf("maximum password length = %d; want 72", hasher.MaxByteLength())
	}
	// Cyrillic characters occupy two bytes: bcrypt's boundary is bytes, not runes.
	password := strings.Repeat("я", 36)
	value, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("72-byte password rejected: %v", err)
	}
	if err := hasher.Compare(password, value); err != nil {
		t.Fatal(err)
	}
	value, err = hasher.Hash(password + "a")
	if !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Errorf("73-byte password: error = %v; want ErrPasswordTooLong", err)
	}
	if value != "" {
		t.Error("failed hashing returned a nonempty hash")
	}
}

func TestBcryptCompareMalformedHash(t *testing.T) {
	hasher := &passwordhash.BcryptHasher{}
	for _, value := range []string{"", "not-a-bcrypt-hash"} {
		err := hasher.Compare("secret", value)
		if !errors.Is(err, bcrypt.ErrHashTooShort) {
			t.Errorf("malformed hash: error = %v; want ErrHashTooShort", err)
		}
		if errors.Is(err, domain.ErrPasswordMismatch) {
			t.Error("malformed hash treated as password mismatch")
		}
	}
}
