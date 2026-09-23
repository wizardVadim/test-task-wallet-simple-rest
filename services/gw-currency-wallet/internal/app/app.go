package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wallet-app/internal/core/config"
	"wallet-app/internal/core/infrastructure/hash"
	"wallet-app/internal/core/infrastructure/id"
	"wallet-app/internal/core/infrastructure/token"
	auth_repository "wallet-app/internal/features/auth/repository"
	auth_service "wallet-app/internal/features/auth/service"
	auth_http "wallet-app/internal/features/auth/transport/http"
	"wallet-app/internal/features/wallet/repository"
	"wallet-app/internal/features/wallet/service"
	wallet_http "wallet-app/internal/features/wallet/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})).With("service", "gw-currency-wallet")
	return RunWithConfig(cfg, logger)
}

func RunWithConfig(config config.Config, logger *slog.Logger) error {
	slog.SetDefault(logger)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("wallet starting")

	connectionString := url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(config.DB.User, config.DB.Password),
		Host:   net.JoinHostPort(config.DB.Host, config.DB.Port),
		Path:   "/" + config.DB.Name,
	}

	poolConfig, err := pgxpool.ParseConfig(connectionString.String())
	if err != nil {
		return errors.New("invalid database connection configuration")
	}

	poolConfig.MaxConns = int32(config.MaxDbConnections)
	poolConfig.MinConns = int32(config.MinDbConnections)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	logger.Debug("database connection established")

	hasher := hash.New()
	tokenGenerator, err := token.NewJWTGenerator(config.Authorization.JwtSecretKey, time.Hour*time.Duration(config.Authorization.JwtTTL))
	if err != nil {
		return fmt.Errorf("%w: invalid token generator config", err)
	}

	walletRepository := repository.NewPostgresRepository(pool)
	authRepository := auth_repository.NewPostgresRepository(pool)

	walletService := service.New(walletRepository, id.GenerateUUID)
	authService := auth_service.New(authRepository, id.GenerateUUID, hasher, tokenGenerator)

	walletHandler := wallet_http.New(walletService)
	authHandler := auth_http.New(authService)

	mux := http.NewServeMux()

	mux.Handle("POST /api/v1/wallets", auth_http.Authenticate(tokenGenerator, http.HandlerFunc(walletHandler.CreateWallet)))
	mux.Handle("GET /api/v1/wallets/{wallet_uuid}", auth_http.Authenticate(tokenGenerator, http.HandlerFunc(walletHandler.GetWalletBalance)))
	mux.Handle("POST /api/v1/wallet", auth_http.Authenticate(tokenGenerator, http.HandlerFunc(walletHandler.ChangeWalletBalance)))
	mux.HandleFunc("POST /api/v1/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/login", authHandler.Login)
	mux.Handle("GET /api/v1/balance", auth_http.Authenticate(tokenGenerator, http.HandlerFunc(walletHandler.GetBalances)))

	server := &http.Server{
		Addr:              ":" + config.HTTPPort,
		Handler:           wallet_http.Logging(logger, mux),
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		ReadHeaderTimeout: time.Duration(config.ReadHeaderTimeout) * time.Second,
		ReadTimeout:       time.Duration(config.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(config.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(config.IdleTimeout) * time.Second,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen http: %w", err)
	}
	defer listener.Close()
	logger.Info("http server listening", "address", listener.Addr().String())

	stopCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server: %w", err)

	case <-stopCtx.Done():
		logger.Info("http shutdown started")
		stop()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("graceful shutdown failed; closing active connections")
			_ = server.Close()
			return fmt.Errorf("shutdown http server: %w", err)
		}

		logger.Info("http shutdown completed")
		return nil
	}
}
