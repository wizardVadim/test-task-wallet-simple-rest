package domain

import "errors"

var (
	ErrInvalidCurrencyType             = errors.New("not valid currency type")
	ErrRateValueBelowEqualsZero        = errors.New("rate value is below or equal zero")
	ErrRateValueIsNaN                  = errors.New("rate value is NaN")
	ErrInvalidRateValueForSameCurrency = errors.New("invalid rate value for same currency; wanted: 1.0")
	ErrRateValueIsInf                  = errors.New("rate value is inf")
)
