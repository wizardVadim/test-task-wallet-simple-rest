package wallet_http

import "encoding/json"

type ErrorMessage string

const (
	ErrorInvalidRequestBody               ErrorMessage = "invalid request body"
	ErrorBalanceOverflow                  ErrorMessage = "balance overflow"
	ErrorRequestBodyTooLarge              ErrorMessage = "request body too large"
	ErrorInternalServerError              ErrorMessage = "internal server error"
	ErrorInvalidCurrency                  ErrorMessage = "invalid currency"
	ErrorInvalidAmountOrCurrency          ErrorMessage = "Invalid amount or currency"
	ErrorInsufficientFundsOrInvalidAmount ErrorMessage = "Insufficient funds or invalid amount"
)

type ErrorResponse struct {
	Error ErrorMessage `json:"error"`
}

type GetBalancesResponse struct {
	Balance map[string]json.Number `json:"balance"`
}

type ApplyBalanceDTO struct {
	Currency string      `json:"currency"`
	Amount   json.Number `json:"amount"`
}

type ApplyBalanceResponse struct {
	Message    string                 `json:"message"`
	NewBalance map[string]json.Number `json:"new_balance"`
}

const MessageApplyBalanceSuccessDeposit = "Account topped up successfully"
const MessageApplyBalanceSuccessWithdraw = "Withdrawal successful"
