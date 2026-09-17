package domain

import "strings"

type CurrencyType string

const (
	CurrencyUSD CurrencyType = "USD"
	CurrencyRUB CurrencyType = "RUB"
	CurrencyEUR CurrencyType = "EUR"
)

const BaseCurrency = CurrencyUSD

type Currency struct {
	currencyType CurrencyType
}

func NewCurrency(currencyType CurrencyType) (Currency, error) {
	currency := Currency{
		currencyType: CurrencyType(strings.ToUpper(string(currencyType))),
	}

	if err := currency.validate(); err != nil {
		return Currency{}, err
	}

	return currency, nil
}

func (currency Currency) validate() error {
	switch currency.currencyType {
	case CurrencyEUR, CurrencyRUB, CurrencyUSD:
		return nil
	default:
		return ErrInvalidCurrencyType
	}
}

func (currency Currency) CurrencyType() CurrencyType {
	return currency.currencyType
}

func (currency Currency) IsEqual(other Currency) bool {
	return currency.currencyType == other.currencyType
}

func (currency Currency) IsValid() bool {
	if err := currency.validate(); err != nil {
		return false
	}
	return true
}
