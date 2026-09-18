package config_test

import (
	"exchanger-app/internal/core/config"
	"strings"
	"testing"
)

func setEnv(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"POSTGRES_HOST_EXCHANGER": "db-exchanger", "POSTGRES_PORT_EXCHANGER": "5432",
		"POSTGRES_USER_EXCHANGER": "test", "POSTGRES_PASSWORD_EXCHANGER": "test-password",
		"POSTGRES_NAME_EXCHANGER": "exchanger", "GRPC_ADDR": "127.0.0.1", "GRPC_PORT": "51051",
		"LOG_LEVEL_EXCHANGER": "INFO",
		"POSTGRES_HOST":       "wallet-db", "POSTGRES_PORT": "5439", "HTTP_PORT": "8181",
	} {
		t.Setenv(k, v)
	}
}

func TestLoad(t *testing.T) {
	setEnv(t)
	got, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	want := config.Config{GRPC: config.GRPCConfig{Addr: "127.0.0.1", Port: "51051"}, DB: config.DBConfig{Host: "db-exchanger", Port: "5432", User: "test", Password: "test-password", Name: "exchanger"}}
	if got != want {
		t.Fatal("configuration did not match exchanger environment")
	}
}

func TestRequiredEnvironment(t *testing.T) {
	for _, key := range []string{"POSTGRES_HOST_EXCHANGER", "POSTGRES_PORT_EXCHANGER", "POSTGRES_USER_EXCHANGER", "POSTGRES_PASSWORD_EXCHANGER", "POSTGRES_NAME_EXCHANGER", "GRPC_ADDR", "GRPC_PORT"} {
		t.Run(key, func(t *testing.T) {
			setEnv(t)
			t.Setenv(key, "")
			got, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("expected error naming %s, got %v", key, err)
			}
			if got != (config.Config{}) {
				t.Fatal("nonempty configuration on failure")
			}
		})
	}
}
