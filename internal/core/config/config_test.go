package config_test

import (
	"fmt"
	"os"
	"testing"

	"wallet-app/internal/core/config"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"HTTP_ADDR":         "8080",
		"POSTGRES_HOST":     "localhost",
		"POSTGRES_PORT":     "5432",
		"POSTGRES_USER":     "test_user",
		"POSTGRES_PASSWORD": "test_password",
		"POSTGRES_NAME":     "wallet_test",
	} {
		t.Setenv(key, value)
	}
}

func TestLoad(t *testing.T) {
	setValidEnv(t)

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := config.Config{
		HTTPAddr: "8080",
		DB: config.DBConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "test_user",
			Password: "test_password",
			Name:     "wallet_test",
		},
	}
	if got != want {
		t.Fatalf("Load() = %+v; want %+v", got, want)
	}
}

func TestLoadRequiredEnv(t *testing.T) {
	keys := []string{
		"HTTP_ADDR",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_NAME",
	}

	for _, key := range keys {
		for _, state := range []string{"empty", "unset"} {
			t.Run(key+"/"+state, func(t *testing.T) {
				setValidEnv(t)
				if state == "empty" {
					t.Setenv(key, "")
				} else {
					if err := os.Unsetenv(key); err != nil {
						t.Fatalf("Unsetenv(%q): %v", key, err)
					}
				}

				got, err := config.Load()
				wantErr := fmt.Sprintf("environment variable %s is required", key)
				if err == nil || err.Error() != wantErr {
					t.Fatalf("Load() error = %v; want %q", err, wantErr)
				}
				if got != (config.Config{}) {
					t.Fatalf("Load() returned nonzero config on error: %+v", got)
				}
			})
		}
	}
}
