package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log/slog"
	"os"
)

type Config struct {
	GRPC     GRPCConfig
	DB       DBConfig
	LogLevel slog.Level
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type GRPCConfig struct {
	Addr string
	Port string
}

// LoadFile reads dotenv values without changing the process environment.
// Explicit environment variables take precedence, including empty values.
func LoadFile(path string) (Config, error) {
	values, err := godotenv.Read(path)
	if err != nil {
		// Parser errors can contain file contents: do not expose secrets in diagnostics.
		return Config{}, fmt.Errorf("cannot read or parse configuration file %q", path)
	}
	return load(func(key string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		return values[key]
	})
}

func Load() (Config, error) { return load(os.Getenv) }

func load(getenv func(string) string) (Config, error) {
	required := []string{
		"POSTGRES_HOST_EXCHANGER",
		"POSTGRES_PORT_EXCHANGER",
		"POSTGRES_USER_EXCHANGER",
		"POSTGRES_PASSWORD_EXCHANGER",
		"POSTGRES_NAME_EXCHANGER",
		"GRPC_ADDR",
		"GRPC_PORT",
	}

	for _, key := range required {
		if getenv(key) == "" {
			return Config{}, fmt.Errorf("environment variable %s is required", key)
		}
	}

	config := Config{
		GRPC: GRPCConfig{
			Addr: getenv("GRPC_ADDR"),
			Port: getenv("GRPC_PORT"),
		},
		DB: DBConfig{
			Host:     getenv("POSTGRES_HOST_EXCHANGER"),
			Port:     getenv("POSTGRES_PORT_EXCHANGER"),
			User:     getenv("POSTGRES_USER_EXCHANGER"),
			Password: getenv("POSTGRES_PASSWORD_EXCHANGER"),
			Name:     getenv("POSTGRES_NAME_EXCHANGER"),
		},
	}

	if level := getenv("LOG_LEVEL_EXCHANGER"); level != "" {
		if err := config.LogLevel.UnmarshalText([]byte(level)); err != nil {
			return Config{}, fmt.Errorf("LOG_LEVEL_EXCHANGER must be DEBUG, INFO, WARN or ERROR")
		}
	}

	return config, nil
}
