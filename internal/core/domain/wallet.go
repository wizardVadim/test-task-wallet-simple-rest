package domain

import (
	"strings"

	"github.com/google/uuid"
)

type WalletID struct {
	value uuid.UUID
}

func NewWalletID(value uuid.UUID) (WalletID, error) {
	walletID := WalletID{value: value}
	if err := walletID.validate(); err != nil {
		return WalletID{}, err
	}

	return walletID, nil
}

func (walletID WalletID) validate() error {
	if walletID.value == uuid.Nil {
		return ErrWalletValueIsEmpty
	}
	return nil
}

func (walletID WalletID) Value() uuid.UUID {
	return walletID.value
}

func (walletID WalletID) IsEqual(other WalletID) bool {
	return walletID.value == other.value
}

type OperationType string

const (
	OperationTypeDeposit  OperationType = "deposit"
	OperationTypeWithdraw OperationType = "withdraw"
)

type WalletOperation struct {
	walletID      WalletID
	operationType OperationType
	amount        int64
}

func (operation WalletOperation) WalletID() WalletID {
	return operation.walletID
}

func (operation WalletOperation) OperationType() OperationType {
	return operation.operationType
}

func (operation WalletOperation) Amount() int64 {
	return operation.amount
}

func NewWalletOperation(walletID WalletID, operationType OperationType, amount int64) (WalletOperation, error) {
	walletOperation := WalletOperation{
		walletID:      walletID,
		operationType: OperationType(strings.ToLower(string(operationType))),
		amount:        amount,
	}
	if err := walletOperation.validate(); err != nil {
		return WalletOperation{}, err
	}
	return walletOperation, nil
}

func (operation WalletOperation) validate() error {
	if err := operation.walletID.validate(); err != nil {
		return err
	}

	if operation.amount <= 0 {
		return ErrAmountMustBePositive
	}

	switch operation.operationType {
	case OperationTypeDeposit, OperationTypeWithdraw:
		return nil
	default:
		return ErrInvalidOperationType
	}
}

type Wallet struct {
	id      WalletID
	balance int64
}

func (wallet Wallet) validate() error {
	if err := wallet.id.validate(); err != nil {
		return err
	}
	if wallet.balance < 0 {
		return ErrBalanceMustBePositiveOrZero
	}
	return nil
}

func (wallet Wallet) ID() WalletID {
	return wallet.id
}

func (wallet Wallet) Balance() int64 {
	return wallet.balance
}

func NewWallet(id WalletID, balance int64) (Wallet, error) {
	wallet := Wallet{id: id, balance: balance}
	if err := wallet.validate(); err != nil {
		return Wallet{}, err
	}

	return wallet, nil
}
