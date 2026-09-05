package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr string
	DB       DBConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func Load() (Config, error) {
	required := []string{
		"HTTP_ADDR",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_NAME",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return Config{}, fmt.Errorf("environment variable %s is required", key)
		}
	}

	return Config{
		HTTPAddr: os.Getenv("HTTP_ADDR"),
		DB: DBConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Name:     os.Getenv("POSTGRES_NAME"),
		},
	}, nil
}
