package wallet_http

type ErrorDTO struct {
	Message ErrorMessage `json:"message"`
}

type ResponseDTO struct {
	Payload any       `json:"payload"`
	Error   *ErrorDTO `json:"error,omitempty"`
}

type ErrorMessage string

const (
	ErrorInvalidWalletID       ErrorMessage = "invalid wallet id"
	ErrorWalletNotFound        ErrorMessage = "wallet not found"
	ErrorGetBalanceInternal    ErrorMessage = "couldn't get wallet balance"
	ErrorCreateWalletInternal  ErrorMessage = "couldn't create a new wallet"
	ErrorInvalidRequestBody    ErrorMessage = "invalid request body"
	ErrorBalanceOverflow       ErrorMessage = "balance overflow"
	ErrorInsufficientFunds     ErrorMessage = "insufficient funds"
	ErrorRequestBodyTooLarge   ErrorMessage = "request body too large"
	ErrorChangeBalanceInternal ErrorMessage = "couldn't change wallet balance"
)

type GetBalancePayload struct {
	Balance int64 `json:"balance"`
}

type CreateWalletPayload struct {
	WalletID string `json:"walletId"`
	Balance  int64  `json:"balance"`
}

type ChangeBalanceDTO struct {
	WalletID      string `json:"walletId"`
	OperationType string `json:"operationType"`
	Amount        int64  `json:"amount"`
}
