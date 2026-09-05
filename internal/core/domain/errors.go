package domain

import "errors"

var (
	ErrWalletValueIsEmpty          = errors.New("wallet value is empty")
	ErrInvalidOperationType        = errors.New("invalid operation type")
	ErrAmountMustBePositive        = errors.New("amount below zero or equals zero")
	ErrBalanceMustBePositiveOrZero = errors.New("balance below zero")
)
