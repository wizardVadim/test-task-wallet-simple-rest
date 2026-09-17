package grpc

import (
	"context"
	"log/slog"
	"time"

	grpcgo "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Logging records one completion event per RPC without request contents or credentials.
func Logging(logger *slog.Logger) grpcgo.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpcgo.UnaryServerInfo, handler grpcgo.UnaryHandler) (any, error) {
		start := time.Now()
		response, err := handler(ctx, req)
		code := status.Code(err)
		level := slog.LevelInfo
		if code == codes.Internal || code == codes.Unknown {
			level = slog.LevelError
		} else if code != codes.OK {
			level = slog.LevelWarn
		}
		logger.Log(ctx, level, "grpc request completed", "method", info.FullMethod, "code", code.String(), "duration_ms", float64(time.Since(start).Microseconds())/1000)
		return response, err
	}
}
