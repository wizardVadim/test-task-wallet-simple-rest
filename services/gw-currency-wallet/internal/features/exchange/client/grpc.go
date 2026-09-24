package client

import (
	"context"
	"contracts/exchange"
	"wallet-app/internal/core/domain"
)

type Client struct {
	exchangeServiceClient exchange.ExchangeServiceClient
}

func New(exchangeServiceClient exchange.ExchangeServiceClient) *Client {
	return &Client{
		exchangeServiceClient: exchangeServiceClient,
	}
}

func (client *Client) GetExchangeRates(ctx context.Context) (domain.ExchangeRates, error) {
	response, err := client.exchangeServiceClient.GetExchangeRates(ctx, &exchange.Empty{})
	if err != nil {
		return domain.ExchangeRates{}, err
	}
	baseCurrency, err := domain.NewCurrency(domain.CurrencyType(response.BaseCurrency))
	if err != nil {
		return domain.ExchangeRates{}, err
	}
	rates := make(domain.Rates, len(response.Rates))
	for k, v := range response.Rates {
		rates[domain.CurrencyType(k)] = v
	}
	exchangeRates, err := domain.NewExchangeRates(baseCurrency, rates)
	if err != nil {
		return domain.ExchangeRates{}, err
	}
	return exchangeRates, nil
}
