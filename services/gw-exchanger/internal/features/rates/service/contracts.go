package service

import (
	"context"
	"exchanger-app/internal/core/domain"
)

type RateRepository interface {
	GetAll(ctx context.Context) ([]domain.ExchangeRate, error)
}
