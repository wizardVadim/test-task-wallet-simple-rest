package domain

import (
	"fmt"

	"github.com/google/uuid"
)

type Balance struct {
	userID   uuid.UUID
	currency Currency
	amount   int64
}

func NewBalance(userID uuid.UUID, currency Currency, amount int64) (Balance, error) {
	balance := Balance{
		userID:   userID,
		currency: currency,
		amount:   amount,
	}

	if err := balance.validate(); err != nil {
		return Balance{}, err
	}

	return balance, nil
}

func (balance Balance) validate() error {
	if balance.userID == uuid.Nil {
		return fmt.Errorf("%w: %+v", ErrInvalidUserID, balance.userID)
	}
	if err := balance.currency.validate(); err != nil {
		return fmt.Errorf("%w: %+v", err, balance.currency)
	}
	if balance.amount < 0 {
		return fmt.Errorf("%w: %+v", ErrInvalidBalanceAmount, balance.amount)
	}
	return nil
}

func (balance Balance) UserID() uuid.UUID {
	return balance.userID
}

func (balance Balance) Currency() Currency {
	return balance.currency
}

func (balance Balance) Amount() int64 {
	return balance.amount
}
