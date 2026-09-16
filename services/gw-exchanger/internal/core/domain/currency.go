package domain

type CurrencyType string

const (
	CurrencyUSD CurrencyType = "USD"
	CurrencyRUB CurrencyType = "RUB"
	CurrencyEUR CurrencyType = "EUR"
)

type Currency struct {
	currencyType CurrencyType
}

func NewCurrency(currencyType CurrencyType) (Currency, error) {
	currency := Currency{
		currencyType: currencyType,
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
