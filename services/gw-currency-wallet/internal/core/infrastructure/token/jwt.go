package token

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type JWTGenerator struct {
	key string
	ttl time.Duration
}

func NewJWTGenerator(key string, ttl time.Duration) (*JWTGenerator, error) {
	g := JWTGenerator{key: key, ttl: ttl}
	if err := g.validateConfig(); err != nil {
		return nil, err
	}
	return &g, nil
}

func (g *JWTGenerator) Generate(subject string) (string, error) {
	if strings.TrimSpace(subject) == "" {
		return "", fmt.Errorf("%w: empty subject", ErrInvalidToken)
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(g.ttl)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(g.key))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (g *JWTGenerator) Validate(tokenString string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (any, error) {
		return []byte(g.key), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if !parsed.Valid || strings.TrimSpace(claims.Subject) == "" {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

func (g *JWTGenerator) validateConfig() error {
	if len(g.key) < 32 {
		return errors.New("JWT signing key must contain at least 32 bytes")
	}
	// JWT NumericDate uses whole seconds by default.
	if g.ttl < time.Second {
		return errors.New("JWT lifetime must be at least one second")
	}
	return nil
}
