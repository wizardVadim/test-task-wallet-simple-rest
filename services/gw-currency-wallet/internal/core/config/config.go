package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr          string
	DB                DBConfig
	MaxDbConnections  int
	MinDbConnections  int
	ReadHeaderTimeout int
	ReadTimeout       int
	WriteTimeout      int
	IdleTimeout       int
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
		"MAX_DB_CONNECTIONS",
		"MIN_DB_CONNECTIONS",
		"READ_HEADER_TIMEOUT",
		"READ_TIMEOUT",
		"WRITE_TIMEOUT",
		"IDLE_TIMEOUT",
	}

	for _, key := range required {
		if os.Getenv(key) == "" {
			return Config{}, fmt.Errorf("environment variable %s is required", key)
		}
	}

	maxDbConnections, err := strconv.Atoi(os.Getenv("MAX_DB_CONNECTIONS"))
	if err != nil {
		return Config{}, fmt.Errorf("MAX_DB_CONNECTIONS: cannot convert variable: %w", err)
	}
	minDbConnections, err := strconv.Atoi(os.Getenv("MIN_DB_CONNECTIONS"))
	if err != nil {
		return Config{}, fmt.Errorf("MIN_DB_CONNECTIONS: cannot convert variable: %w", err)
	}
	readHeaderTimeout, err := strconv.Atoi(os.Getenv("READ_HEADER_TIMEOUT"))
	if err != nil {
		return Config{}, fmt.Errorf("READ_HEADER_TIMEOUT: cannot convert variable: %w", err)
	}
	readTimeout, err := strconv.Atoi(os.Getenv("READ_TIMEOUT"))
	if err != nil {
		return Config{}, fmt.Errorf("READ_TIMEOUT: cannot convert variable: %w", err)
	}
	writeTimeout, err := strconv.Atoi(os.Getenv("WRITE_TIMEOUT"))
	if err != nil {
		return Config{}, fmt.Errorf("WRITE_TIMEOUT: cannot convert variable: %w", err)
	}
	idleTimeout, err := strconv.Atoi(os.Getenv("IDLE_TIMEOUT"))
	if err != nil {
		return Config{}, fmt.Errorf("IDLE_TIMEOUT: cannot convert variable: %w", err)
	}

	config := Config{
		HTTPAddr:          os.Getenv("HTTP_ADDR"),
		MaxDbConnections:  maxDbConnections,
		MinDbConnections:  minDbConnections,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		DB: DBConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Name:     os.Getenv("POSTGRES_NAME"),
		},
	}

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (config Config) validate() error {
	if config.MaxDbConnections < 1 || config.MaxDbConnections > math.MaxInt32 {
		return fmt.Errorf("MAX_DB_CONNECTIONS error: %v", errInvalidSettingValue)
	}
	if config.MinDbConnections < 0 || config.MinDbConnections > config.MaxDbConnections {
		return fmt.Errorf("MIN_DB_CONNECTIONS error: %v", errInvalidSettingValue)
	}
	if config.ReadHeaderTimeout < 1 || int64(config.ReadHeaderTimeout) > math.MaxInt64/int64(time.Second) {
		return fmt.Errorf("READ_HEADER_TIMEOUT error: %v", errInvalidSettingValue)
	}
	if config.ReadTimeout < 1 || int64(config.ReadTimeout) > math.MaxInt64/int64(time.Second) {
		return fmt.Errorf("READ_TIMEOUT error: %v", errInvalidSettingValue)
	}
	if config.WriteTimeout < 1 || int64(config.WriteTimeout) > math.MaxInt64/int64(time.Second) {
		return fmt.Errorf("WRITE_TIMEOUT error: %v", errInvalidSettingValue)
	}
	if config.IdleTimeout < 1 || int64(config.IdleTimeout) > math.MaxInt64/int64(time.Second) {
		return fmt.Errorf("IDLE_TIMEOUT error: %v", errInvalidSettingValue)
	}
	return nil
}
