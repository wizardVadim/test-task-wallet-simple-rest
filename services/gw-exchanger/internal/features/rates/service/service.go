package service

import (
	"context"
	"exchanger-app/internal/core/domain"
	"fmt"
)

type Service struct {
	repository RateRepository
}

func New(repository RateRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (service *Service) GetAll(ctx context.Context) ([]domain.ExchangeRate, error) {
	if err := ctx.Err(); err != nil {
		return []domain.ExchangeRate{}, err
	}
	return service.repository.GetAll(ctx)
}

func (service *Service) GetRate(ctx context.Context, from domain.Currency, to domain.Currency) (domain.ExchangeRate, error) {
	if err := ctx.Err(); err != nil {
		return domain.ExchangeRate{}, err
	}
	if !from.IsValid() {
		return domain.ExchangeRate{}, fmt.Errorf("currency from error: %w", domain.ErrInvalidCurrencyType)
	}
	if !to.IsValid() {
		return domain.ExchangeRate{}, fmt.Errorf("currency to error: %w", domain.ErrInvalidCurrencyType)
	}
	baseRates, err := service.repository.GetAll(ctx)
	if err != nil {
		return domain.ExchangeRate{}, fmt.Errorf("repository error: %w", err)
	}

	indexes := make(map[string]float32)
	for _, v := range baseRates {
		indexes[string(v.ToCurrency().CurrencyType())] = v.Rate().Value()
	}

	fromValue, ok := indexes[string(from.CurrencyType())]
	if !ok {
		return domain.ExchangeRate{}, fmt.Errorf(
			"%w: %s",
			ErrNotFoundCurrency,
			from.CurrencyType(),
		)
	}
	toValue, ok := indexes[string(to.CurrencyType())]
	if !ok {
		return domain.ExchangeRate{}, fmt.Errorf(
			"%w: %s",
			ErrNotFoundCurrency,
			to.CurrencyType(),
		)
	}

	unitsPerFromCurrency := toValue / fromValue

	rate, err := domain.NewRate(unitsPerFromCurrency)
	if err != nil {
		return domain.ExchangeRate{}, err
	}

	exchangeRate, err := domain.NewExchangeRate(rate, from, to)
	if err != nil {
		return domain.ExchangeRate{}, err
	}

	return exchangeRate, nil
}
