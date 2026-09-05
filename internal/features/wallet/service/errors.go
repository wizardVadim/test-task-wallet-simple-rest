package service

import "errors"

var (
	ErrSmallBalance    = errors.New("insufficient funds on the balance")
	ErrBalanceOverflow = errors.New("balance overflow")
)
