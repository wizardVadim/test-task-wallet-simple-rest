package grpc

import (
	"context"
	"errors"

	"exchanger-app/internal/core/domain"
	"exchanger-app/internal/features/rates/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")

	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")

	case errors.Is(err, domain.ErrInvalidCurrencyType):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, service.ErrNotFoundCurrency):
		return status.Error(codes.NotFound, err.Error())

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
