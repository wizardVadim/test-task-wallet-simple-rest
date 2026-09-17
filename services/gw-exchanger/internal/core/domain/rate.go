package domain

import (
	"errors"
	"fmt"
	"math"
)

type Rate struct {
	value float32
}

func NewRate(value float32) (Rate, error) {
	rate := Rate{
		value: value,
	}

	if err := rate.validate(); err != nil {
		return Rate{}, err
	}

	return rate, nil
}

func (rate Rate) validate() error {
	if math.IsNaN(float64(rate.value)) {
		return ErrRateValueIsNaN
	}
	if math.IsInf(float64(rate.value), 0) {
		return ErrRateValueIsInf
	}
	if rate.value <= 0.0 {
		return ErrRateValueBelowEqualsZero
	}

	return nil
}

func (rate Rate) Value() float32 {
	return rate.value
}

type ExchangeRate struct {
	rate         Rate
	fromCurrency Currency
	toCurrency   Currency
}

func NewExchangeRate(rate Rate, fromCurrency Currency, toCurrency Currency) (ExchangeRate, error) {
	exchangeRate := ExchangeRate{
		rate:         rate,
		fromCurrency: fromCurrency,
		toCurrency:   toCurrency,
	}

	if err := exchangeRate.validate(); err != nil {
		return ExchangeRate{}, err
	}

	return exchangeRate, nil
}

func (exchangeRate ExchangeRate) validate() error {
	var returnErr error
	if err := exchangeRate.rate.validate(); err != nil {
		returnErr = errors.Join(returnErr, err)
	}
	if err := exchangeRate.fromCurrency.validate(); err != nil {
		returnErr = errors.Join(returnErr, fmt.Errorf("error with from currency: %w", err))
	}
	if err := exchangeRate.toCurrency.validate(); err != nil {
		returnErr = errors.Join(returnErr, fmt.Errorf("error with to currency: %w", err))
	}
	if exchangeRate.fromCurrency.IsEqual(exchangeRate.toCurrency) && exchangeRate.rate.value != 1 {
		returnErr = errors.Join(returnErr, ErrInvalidRateValueForSameCurrency)
	}
	return returnErr
}

func (exchangeRate ExchangeRate) Rate() Rate {
	return exchangeRate.rate
}

func (exchangeRate ExchangeRate) FromCurrency() Currency {
	return exchangeRate.fromCurrency
}

func (exchangeRate ExchangeRate) ToCurrency() Currency {
	return exchangeRate.toCurrency
}
