package domain

import (
	"fmt"

	"github.com/google/uuid"
)

type BalanceOperation struct {
	userID        uuid.UUID
	currency      Currency
	operationType OperationType
	amount        int64
}

func NewBalanceOperation(userID uuid.UUID, currency Currency, operationType OperationType, amount int64) (BalanceOperation, error) {
	bOperation := BalanceOperation{
		userID:        userID,
		currency:      currency,
		operationType: operationType,
		amount:        amount,
	}

	if err := bOperation.validate(); err != nil {
		return BalanceOperation{}, err
	}

	return bOperation, nil
}

func (bOperation BalanceOperation) validate() error {
	if bOperation.userID == uuid.Nil {
		return fmt.Errorf("%w: %+v", ErrInvalidUserID, bOperation.userID)
	}
	if err := bOperation.currency.validate(); err != nil {
		return fmt.Errorf("%w: %+v", err, bOperation.currency)
	}
	if bOperation.amount <= 0 {
		return fmt.Errorf("%w: %+v; must be more zero", ErrInvalidBalanceAmount, bOperation.amount)
	}

	switch bOperation.operationType {
	case OperationTypeDeposit, OperationTypeWithdraw:
		return nil
	default:
		return fmt.Errorf("%w: %+v", ErrInvalidOperationType, bOperation.operationType)
	}
}

func (bOperation BalanceOperation) UserID() uuid.UUID {
	return bOperation.userID
}

func (bOperation BalanceOperation) Currency() Currency {
	return bOperation.currency
}

func (bOperation BalanceOperation) Amount() int64 {
	return bOperation.amount
}

func (bOperation BalanceOperation) OperationType() OperationType {
	return bOperation.operationType
}

type OperationType string

const (
	OperationTypeDeposit  OperationType = "deposit"
	OperationTypeWithdraw OperationType = "withdraw"
)

func (operationType OperationType) IsEqual(other OperationType) bool {
	return operationType == other
}
