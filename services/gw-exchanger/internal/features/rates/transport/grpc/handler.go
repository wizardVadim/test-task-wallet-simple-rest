package grpc

import (
	"context"
	"contracts/exchange"
	"exchanger-app/internal/core/domain"
)

type Handler struct {
	exchange.UnimplementedExchangeServiceServer

	service RateService
}

func New(service RateService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetExchangeRates(ctx context.Context, _ *exchange.Empty) (*exchange.ExchangeRatesResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, mapError(err)
	}

	rates, err := h.service.GetAll(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	responseMap := make(map[string]float32, len(rates))
	for _, v := range rates {
		responseMap[string(v.ToCurrency().CurrencyType())] = v.Rate().Value()
	}

	return &exchange.ExchangeRatesResponse{
		Rates:        responseMap,
		BaseCurrency: string(domain.BaseCurrency),
	}, nil
}

func (h *Handler) GetExchangeRateForCurrency(ctx context.Context, req *exchange.CurrencyRequest) (*exchange.ExchangeRateResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, mapError(err)
	}

	from, err := domain.NewCurrency(domain.CurrencyType(req.GetFromCurrency()))
	if err != nil {
		return nil, mapError(err)
	}

	to, err := domain.NewCurrency(domain.CurrencyType(req.GetToCurrency()))
	if err != nil {
		return nil, mapError(err)
	}

	rate, err := h.service.GetRate(ctx, from, to)
	if err != nil {
		return nil, mapError(err)
	}

	response := exchange.ExchangeRateResponse{
		FromCurrency: string(rate.FromCurrency().CurrencyType()),
		ToCurrency:   string(rate.ToCurrency().CurrencyType()),
		Rate:         rate.Rate().Value(),
	}

	return &response, nil
}
