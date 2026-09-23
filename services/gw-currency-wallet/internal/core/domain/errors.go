package domain

import "errors"

var (
	ErrWalletValueIsEmpty          = errors.New("wallet value is empty")
	ErrInvalidOperationType        = errors.New("invalid operation type")
	ErrAmountMustBePositive        = errors.New("amount below zero or equals zero")
	ErrBalanceMustBePositiveOrZero = errors.New("balance below zero")
	ErrWalletNotFound              = errors.New("wallet not found")
)

var (
	ErrSmallBalance    = errors.New("insufficient funds on the balance")
	ErrBalanceOverflow = errors.New("balance overflow")
)

var (
	ErrInvalidUsername        = errors.New("invalid username")
	ErrInvalidEmailAddress    = errors.New("invalid email address")
	ErrInvalidUserID          = errors.New("invalid user id")
	ErrInvalidPasswordHash    = errors.New("invalid password hash")
	ErrUserNotFound           = errors.New("user is not found")
	ErrUsernameAlreadyExists  = errors.New("username already exists")
	ErrEmailAlreadyExists     = errors.New("email already exists")
	ErrInvalidPassword        = errors.New("invalid password")
	ErrInvalidUserCredentials = errors.New("invalid user credentials")
	ErrPasswordMismatch       = errors.New("password mismatch")
)

var (
	ErrInvalidCurrencyType  = errors.New("invalid currency type")
	ErrInvalidBalanceAmount = errors.New("invalid balance amount")
)
