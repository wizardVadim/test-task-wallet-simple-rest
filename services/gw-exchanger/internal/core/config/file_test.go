package config_test

import (
	"exchanger-app/internal/core/config"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clearExchangerEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"POSTGRES_HOST_EXCHANGER", "POSTGRES_PORT_EXCHANGER", "POSTGRES_USER_EXCHANGER", "POSTGRES_PASSWORD_EXCHANGER", "POSTGRES_NAME_EXCHANGER", "GRPC_ADDR", "GRPC_PORT", "LOG_LEVEL_EXCHANGER"} {
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

const fileConfig = `# local exchanger
POSTGRES_HOST_EXCHANGER=db-exchanger
POSTGRES_PORT_EXCHANGER=5431
POSTGRES_USER_EXCHANGER=test
POSTGRES_PASSWORD_EXCHANGER='password with # spaces'
POSTGRES_NAME_EXCHANGER=exchanger
GRPC_ADDR=127.0.0.1
GRPC_PORT=50051
LOG_LEVEL_EXCHANGER=DEBUG
`

func TestLoadFileAndEnvironmentPrecedence(t *testing.T) {
	clearExchangerEnv(t)
	path := writeConfig(t, fileConfig)
	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Host != "db-exchanger" || cfg.DB.Password != "password with # spaces" || cfg.LogLevel != slog.LevelDebug {
		t.Fatal("file not read correctly")
	}
	if _, ok := os.LookupEnv("GRPC_PORT"); ok {
		t.Fatal("LoadFile mutated environment")
	}
	t.Setenv("POSTGRES_HOST_EXCHANGER", "localhost")
	t.Setenv("GRPC_PORT", "51051")
	cfg, err = config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.Host != "localhost" || cfg.GRPC.Port != "51051" {
		t.Fatal("environment did not override file")
	}
	t.Setenv("GRPC_PORT", "")
	if _, err = config.LoadFile(path); err == nil {
		t.Fatal("explicit empty environment must not fall back to file")
	}
}
func TestLoadFileFailures(t *testing.T) {
	clearExchangerEnv(t)
	for _, path := range []string{filepath.Join(t.TempDir(), "missing.env"), writeConfig(t, "PASSWORD='private-secret\n")} {
		if _, err := config.LoadFile(path); err == nil || strings.Contains(err.Error(), "private-secret") {
			t.Fatalf("expected safe configuration error, got %v", err)
		}
	}
}
func TestLogLevel(t *testing.T) {
	for _, level := range []string{"DEBUG", "INFO", "WARN", "ERROR", "invalid"} {
		t.Run(level, func(t *testing.T) {
			setEnv(t)
			t.Setenv("LOG_LEVEL_EXCHANGER", level)
			_, err := config.Load()
			if (err != nil) != (level == "invalid") {
				t.Fatalf("level %s: %v", level, err)
			}
		})
	}
}
