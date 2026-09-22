package token_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"wallet-app/internal/core/infrastructure/token"
	"wallet-app/internal/features/auth/service"
)

const key = "0123456789abcdef0123456789abcdef"

var _ service.JWTGenerator = (*token.JWTGenerator)(nil)

func TestGenerateAndValidate(t *testing.T) {
	generator, err := token.NewJWTGenerator(key, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now().Unix()
	encoded, err := generator.Generate("external-user-42")
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now().Unix()
	subject, err := generator.Validate(encoded)
	if err != nil || subject != "external-user-42" {
		t.Fatalf("subject = %q, error = %v", subject, err)
	}
	claims := &jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(encoded, claims, func(*jwt.Token) (any, error) { return []byte(key), nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		t.Fatal(err)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatal("missing time claims")
	}
	if claims.IssuedAt.Unix() < before || claims.IssuedAt.Unix() > after {
		t.Error("incorrect issuance time")
	}
	if claims.ExpiresAt.Unix()-claims.IssuedAt.Unix() != 3600 {
		t.Error("incorrect lifetime")
	}
}

func TestValidateRejectsInvalidTokens(t *testing.T) {
	generator, err := token.NewJWTGenerator(key, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, subject     string
		expiry, notBefore *jwt.NumericDate
		method            jwt.SigningMethod
		signingKey        string
	}{
		{name: "expired", subject: "user", expiry: jwt.NewNumericDate(time.Now().Add(-time.Hour))},
		{name: "missing expiry", subject: "user"},
		{name: "empty subject", expiry: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		{name: "blank subject", subject: " \t", expiry: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		{name: "wrong key", subject: "user", expiry: jwt.NewNumericDate(time.Now().Add(time.Hour)), signingKey: strings.Repeat("x", 32)},
		{name: "wrong algorithm", subject: "user", expiry: jwt.NewNumericDate(time.Now().Add(time.Hour)), method: jwt.SigningMethodHS384},
		{name: "not yet valid", subject: "user", expiry: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)), notBefore: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			method := tt.method
			if method == nil {
				method = jwt.SigningMethodHS256
			}
			signingKey := tt.signingKey
			if signingKey == "" {
				signingKey = key
			}
			encoded, err := jwt.NewWithClaims(method, jwt.RegisteredClaims{Subject: tt.subject, ExpiresAt: tt.expiry, NotBefore: tt.notBefore}).SignedString([]byte(signingKey))
			if err != nil {
				t.Fatal(err)
			}
			subject, err := generator.Validate(encoded)
			if subject != "" || !errors.Is(err, token.ErrInvalidToken) {
				t.Fatalf("subject = %q, error = %v; want invalid token", subject, err)
			}
		})
	}
	for _, encoded := range []string{"", "garbage", "a.b.c", "eyJhbGciOiJub25lIn0.eyJzdWIiOiJ1c2VyIn0."} {
		if subject, err := generator.Validate(encoded); subject != "" || !errors.Is(err, token.ErrInvalidToken) {
			t.Errorf("malformed token accepted: %v", err)
		}
	}
}

func TestGenerateRejectsEmptySubject(t *testing.T) {
	generator, err := token.NewJWTGenerator(key, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for _, subject := range []string{"", " \t\n"} {
		encoded, err := generator.Generate(subject)
		if encoded != "" || !errors.Is(err, token.ErrInvalidToken) {
			t.Errorf("empty subject accepted: %v", err)
		}
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name, key string
		ttl       time.Duration
	}{
		{"empty key", "", time.Hour}, {"short key", strings.Repeat("x", 31), time.Hour},
		{"zero lifetime", key, 0}, {"negative lifetime", key, -time.Hour}, {"subsecond lifetime", key, time.Millisecond},
	} {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := token.NewJWTGenerator(tt.key, tt.ttl)
			if err == nil {
				t.Fatal("invalid configuration accepted by constructor")
			}
			if generator != nil {
				t.Error("constructor returned a generator for invalid configuration")
			}
			if errors.Is(err, token.ErrInvalidToken) {
				t.Error("configuration error must be distinct from invalid token")
			}
		})
	}
}

func TestConstructorAcceptsMinimumConfiguration(t *testing.T) {
	generator, err := token.NewJWTGenerator(key, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if generator == nil {
		t.Fatal("constructor returned nil for valid configuration")
	}
}
