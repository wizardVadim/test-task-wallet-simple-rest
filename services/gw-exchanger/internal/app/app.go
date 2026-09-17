package app

import (
	"context"
	"errors"
	"exchanger-app/internal/core/config"
	"exchanger-app/internal/features/rates/repository"
	"exchanger-app/internal/features/rates/service"
	ratesgrpc "exchanger-app/internal/features/rates/transport/grpc"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"contracts/exchange"

	"github.com/jackc/pgx/v5/pgxpool"

	googlegrpc "google.golang.org/grpc"
)

const gracefulShutdownTimeout = 5 * time.Second

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})).With("service", "gw-exchanger")
	return RunWithConfig(cfg, logger)
}

func RunWithConfig(config config.Config, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("exchanger starting")

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

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	logger.Debug("database connection established")

	exchangeRepository := repository.NewPostgresRepository(pool)

	exchangeService := service.New(exchangeRepository)

	exchangeHandler := ratesgrpc.New(exchangeService)

	listener, err := net.Listen(
		"tcp",
		net.JoinHostPort(config.GRPC.Addr, config.GRPC.Port),
	)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}
	defer listener.Close()

	grpcServer := googlegrpc.NewServer(googlegrpc.UnaryInterceptor(ratesgrpc.Logging(logger)))

	stopCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErr := make(chan error, 1)

	exchange.RegisterExchangeServiceServer(
		grpcServer,
		exchangeHandler,
	)

	logger.Info("grpc server listening", "address", listener.Addr().String())

	go func() {
		serverErr <- grpcServer.Serve(listener)
	}()

	select {
	case err := <-serverErr:
		if errors.Is(err, googlegrpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("grpc server: %w", err)

	case <-stopCtx.Done():
		logger.Info("grpc shutdown started")
		gracefulDone := make(chan struct{})

		go func() {
			grpcServer.GracefulStop()
			close(gracefulDone)
		}()

		select {
		case <-gracefulDone:
			logger.Info("grpc shutdown completed")
			return nil

		case <-time.After(gracefulShutdownTimeout):
			logger.Warn("graceful shutdown timed out; stopping active requests")
			grpcServer.Stop()
			return nil
		}
	}

}
