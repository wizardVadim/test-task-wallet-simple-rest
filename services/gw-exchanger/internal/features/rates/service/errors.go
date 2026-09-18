package service

import "errors"

var (
	ErrNotFoundCurrency = errors.New("current currency is not found in indexes")
)
