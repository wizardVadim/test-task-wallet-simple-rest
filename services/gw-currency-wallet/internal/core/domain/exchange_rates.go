package domain

import (
	"maps"
	"math"
)

type Rates map[CurrencyType]float32

type ExchangeRates struct {
	baseCurrency Currency
	rates        Rates
}

const BaseCurrencyType = CurrencyTypeUSD

func NewExchangeRates(baseCurrency Currency, rates Rates) (ExchangeRates, error) {
	exchangeRates := ExchangeRates{
		baseCurrency: baseCurrency,
		rates:        maps.Clone(rates),
	}

	if err := exchangeRates.validate(); err != nil {
		return ExchangeRates{}, err
	}
	return exchangeRates, nil
}

func (exchangeRates ExchangeRates) validate() error {
	if err := exchangeRates.baseCurrency.validate(); err != nil {
		return err
	}
	if exchangeRates.rates == nil {
		return ErrExchangeRatesNil
	}
	if exchangeRates.baseCurrency.CurrencyType() != BaseCurrencyType {
		return ErrBaseCurrencyNotConsists
	}
	value, ok := exchangeRates.rates[BaseCurrencyType]
	if !ok {
		return ErrBaseCurrencyNotConsists
	}
	if value != 1 {
		return ErrInvalidBaseCurrencyRate
	}

	for k, v := range exchangeRates.rates {
		switch k {
		case CurrencyTypeEUR, CurrencyTypeRUB, CurrencyTypeUSD:
		default:
			return ErrInvalidCurrencyType
		}
		if v <= 0 || math.IsInf(float64(v), 0) || math.IsNaN(float64(v)) {
			return ErrInvalidExchangeRateValue
		}
	}
	return nil
}

func (exchangeRates ExchangeRates) BaseCurrency() Currency {
	return exchangeRates.baseCurrency
}

func (exchangeRates ExchangeRates) Rates() Rates {
	return maps.Clone(exchangeRates.rates)
}
