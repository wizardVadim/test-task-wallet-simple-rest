package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wallet-app/internal/core/config"
	"wallet-app/internal/features/wallet/repository"
	"wallet-app/internal/features/wallet/service"
	wallet_http "wallet-app/internal/features/wallet/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	config, err := config.Load()
	if err != nil {
		return err
	}

	connectionString := url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(config.DB.User, config.DB.Password),
		Host:   net.JoinHostPort(config.DB.Host, config.DB.Port),
		Path:   "/" + config.DB.Name,
	}

	poolConfig, err := pgxpool.ParseConfig(connectionString.String())
	if err != nil {
		return err
	}

	poolConfig.MaxConns = 50
	poolConfig.MinConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	walletRepository := repository.NewPostgresRepository(pool)

	walletService := service.New(walletRepository, service.GenerateWalletID)

	walletHandler := wallet_http.New(walletService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/wallets", walletHandler.CreateWallet)
	mux.HandleFunc("GET /api/v1/wallets/{wallet_uuid}", walletHandler.GetWalletBalance)
	mux.HandleFunc("POST /api/v1/wallet", walletHandler.ChangeWalletBalance)

	server := &http.Server{
		Addr:              ":" + config.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server: %w", err)

	case <-stopCtx.Done():
		stop()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown http server: %w", err)
		}

		return nil
	}
}
