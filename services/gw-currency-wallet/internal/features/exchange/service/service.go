package service

import (
	"context"
	"wallet-app/internal/core/domain"
)

type Service struct {
	ratesProvider RatesProvider
}

func New(ratesProvider RatesProvider) *Service {
	return &Service{
		ratesProvider: ratesProvider,
	}
}

func (s *Service) GetExchangeRates(ctx context.Context) (domain.ExchangeRates, error) {
	if err := ctx.Err(); err != nil {
		return domain.ExchangeRates{}, err
	}
	return s.ratesProvider.GetExchangeRates(ctx)
}
