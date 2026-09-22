package config_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"wallet-app/internal/core/config"
)

func clearWalletEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_NAME", "HTTP_PORT", "MAX_DB_CONNECTIONS", "MIN_DB_CONNECTIONS", "READ_HEADER_TIMEOUT", "READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT", "LOG_LEVEL_WALLET", "JWT_SECRET_KEY", "JWT_TTL"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}
func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.env")
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

const fileConfig = `# local wallet
POSTGRES_HOST=db-currency-wallet
POSTGRES_PORT=5432
POSTGRES_USER=test
POSTGRES_PASSWORD='password with # spaces'
POSTGRES_NAME=bank
HTTP_PORT=8080
MAX_DB_CONNECTIONS=4
MIN_DB_CONNECTIONS=1
READ_HEADER_TIMEOUT=5
READ_TIMEOUT=5
WRITE_TIMEOUT=20
IDLE_TIMEOUT=120
LOG_LEVEL_WALLET=DEBUG
JWT_SECRET_KEY=0123456789abcdef0123456789abcdef
JWT_TTL=24
`

func TestLoadFileAndEnvironmentPrecedence(t *testing.T) {
	clearWalletEnv(t)
	path := writeConfig(t, fileConfig)
	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Host != "db-currency-wallet" || cfg.DB.Password != "password with # spaces" || cfg.LogLevel != slog.LevelDebug {
		t.Fatal("file not read correctly")
	}
	if _, ok := os.LookupEnv("HTTP_PORT"); ok {
		t.Fatal("LoadFile mutated environment")
	}
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("HTTP_PORT", "8181")
	cfg, err = config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Host != "localhost" || cfg.HTTPPort != "8181" {
		t.Fatal("environment did not override file")
	}
	t.Setenv("HTTP_PORT", "")
	if _, err = config.LoadFile(path); err == nil {
		t.Fatal("explicit empty environment must not fall back to file")
	}
}
func TestLoadFileFailures(t *testing.T) {
	clearWalletEnv(t)
	for _, path := range []string{filepath.Join(t.TempDir(), "missing.env"), writeConfig(t, "PASSWORD='private-secret\n")} {
		if _, err := config.LoadFile(path); err == nil || strings.Contains(err.Error(), "private-secret") {
			t.Fatalf("expected safe configuration error, got %v", err)
		}
	}
}
func TestLogLevel(t *testing.T) {
	for _, level := range []string{"DEBUG", "INFO", "WARN", "ERROR", "invalid"} {
		t.Run(level, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv("LOG_LEVEL_WALLET", level)
			_, err := config.Load()
			if (err != nil) != (level == "invalid") {
				t.Fatalf("level %s: %v", level, err)
			}
		})
	}
}

func TestJWTFileAndEnvironmentPrecedence(t *testing.T) {
	clearWalletEnv(t)
	path := writeConfig(t, fileConfig)
	got, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Authorization.JwtTTL != 24 || got.Authorization.JwtSecretKey != "0123456789abcdef0123456789abcdef" {
		t.Fatal("JWT file values not loaded")
	}
	for _, key := range []string{"JWT_SECRET_KEY", "JWT_TTL"} {
		if _, ok := os.LookupEnv(key); ok {
			t.Errorf("LoadFile mutated %s", key)
		}
	}
	t.Setenv("JWT_SECRET_KEY", strings.Repeat("x", 32))
	t.Setenv("JWT_TTL", "2")
	got, err = config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Authorization.JwtTTL != 2 || got.Authorization.JwtSecretKey != strings.Repeat("x", 32) {
		t.Fatal("environment did not override JWT file values")
	}
	for _, key := range []string{"JWT_SECRET_KEY", "JWT_TTL"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "")
			if _, err := config.LoadFile(path); err == nil || !strings.Contains(err.Error(), key) {
				t.Errorf("explicit empty %s must fail: %v", key, err)
			}
		})
	}
}
