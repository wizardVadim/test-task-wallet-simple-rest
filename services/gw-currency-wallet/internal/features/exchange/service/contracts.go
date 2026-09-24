package service

import (
	"context"
	"wallet-app/internal/core/domain"
)

type RatesProvider interface {
	GetExchangeRates(ctx context.Context) (domain.ExchangeRates, error)
}
