package grpc

import (
	"context"
	"exchanger-app/internal/core/domain"
)

type RateService interface {
	GetAll(ctx context.Context) ([]domain.ExchangeRate, error)
	GetRate(
		ctx context.Context,
		from domain.Currency,
		to domain.Currency,
	) (domain.ExchangeRate, error)
}
