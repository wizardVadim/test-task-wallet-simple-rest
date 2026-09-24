package wallet_http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"wallet-app/internal/core/domain"
	auth_http "wallet-app/internal/features/auth/transport/http"
)

type Handler struct {
	walletService WalletService
}

func New(walletService WalletService) *Handler {
	return &Handler{walletService: walletService}
}

func writeJSON(w http.ResponseWriter, status int, response any) {
	body, err := json.Marshal(response)
	if err != nil {
		slog.Error("marshal response failed", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		slog.Warn("write response failed", "error", err)
	}
}

// GET /api/v1/balance
func (h *Handler) GetBalances(w http.ResponseWriter, r *http.Request) {
	if err := r.Context().Err(); err != nil {
		return
	}

	userID, ok := auth_http.UserIDFromContext(r.Context())
	if !ok {
		if r.Context().Err() != nil {
			return
		}
		slog.ErrorContext(r.Context(), "get balances failed: couldn't find user ID in context", "error", domain.ErrInvalidUserID)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: ErrorInternalServerError})
		return
	}

	balances, err := h.walletService.GetBalances(r.Context(), userID)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		slog.ErrorContext(r.Context(), "get balances failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: ErrorInternalServerError})
		return
	}

	var response GetBalancesResponse
	response.Balance = make(map[string]json.Number)
	for _, v := range balances {
		amount := v.Amount()
		response.Balance[string(v.Currency().CurrencyType())] = amountInt64ToJsonNumber(amount)
	}

	writeJSON(w, http.StatusOK, response)
}

// POST /api/v1/wallet/deposit
//
// request body:
//
//	{
//		"currency": "USD",
//		"amount": 1000.00
//	}
func (h *Handler) BalanceDeposit(w http.ResponseWriter, r *http.Request) {
	h.applyBalanceOperation(w, r, domain.OperationTypeDeposit)
}

// POST /api/v1/wallet/withdraw
//
// request body:
//
//	{
//		"currency": "USD",
//		"amount": 1000.00
//	}
func (h *Handler) BalanceWithdraw(w http.ResponseWriter, r *http.Request) {
	h.applyBalanceOperation(w, r, domain.OperationTypeWithdraw)
}

func (h *Handler) applyBalanceOperation(w http.ResponseWriter, r *http.Request, operationType domain.OperationType) {
	if err := r.Context().Err(); err != nil {
		return
	}

	userID, ok := auth_http.UserIDFromContext(r.Context())
	if !ok {
		if r.Context().Err() != nil {
			return
		}
		slog.ErrorContext(r.Context(), "change balance failed: couldn't find user ID in context", "error", domain.ErrInvalidUserID, "operation", operationType)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: ErrorInternalServerError})
		return
	}

	var applyBalanceDTO ApplyBalanceDTO

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&applyBalanceDTO)
	if err == nil {
		var extra any
		if nextErr := decoder.Decode(&extra); nextErr != io.EOF {
			if nextErr == nil {
				err = errors.New("multiple JSON values")
			} else {
				err = nextErr
			}
		}
	}
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		slog.WarnContext(r.Context(), "change balance failed", "error", err, "operation", operationType)
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{Error: ErrorRequestBodyTooLarge})
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: ErrorInvalidRequestBody})
		return
	}

	amount, err := jsonNumberToAmountInt64(applyBalanceDTO.Amount)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		slog.WarnContext(r.Context(), "change balance failed", "error", err, "operation", operationType)
		var errorMessage ErrorMessage
		if operationType == domain.OperationTypeDeposit {
			errorMessage = ErrorInvalidAmountOrCurrency
		} else {
			errorMessage = ErrorInsufficientFundsOrInvalidAmount
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: errorMessage})
		return
	}

	currency, err := domain.NewCurrency(domain.CurrencyType(applyBalanceDTO.Currency))
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		var errorMessage ErrorMessage
		if operationType == domain.OperationTypeDeposit {
			errorMessage = ErrorInvalidAmountOrCurrency
		} else {
			errorMessage = ErrorInvalidCurrency
		}
		slog.WarnContext(r.Context(), "change balance failed", "error", err, "operation", operationType)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: errorMessage})
		return
	}

	balanceOperation, err := domain.NewBalanceOperation(
		userID,
		currency,
		operationType,
		amount,
	)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		if errors.Is(err, domain.ErrInvalidBalanceAmount) {
			slog.WarnContext(r.Context(), "change balance failed", "error", err, "operation", operationType)
			var errorMessage ErrorMessage
			if operationType == domain.OperationTypeDeposit {
				errorMessage = ErrorInvalidAmountOrCurrency
			} else {
				errorMessage = ErrorInsufficientFundsOrInvalidAmount
			}
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: errorMessage})
			return
		}
		slog.ErrorContext(r.Context(), "change balance failed", "error", err, "operation", operationType)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: ErrorInternalServerError})
		return
	}

	balances, err := h.walletService.ApplyBalanceOperation(r.Context(), balanceOperation)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		var response ErrorResponse
		var statusCode int

		if errors.Is(err, domain.ErrSmallBalance) {
			response.Error = ErrorInsufficientFundsOrInvalidAmount
			statusCode = http.StatusBadRequest
		} else if errors.Is(err, domain.ErrBalanceOverflow) {
			response.Error = ErrorBalanceOverflow
			statusCode = http.StatusUnprocessableEntity
		} else {
			response.Error = ErrorInternalServerError
			statusCode = http.StatusInternalServerError
		}

		level := slog.LevelWarn
		if statusCode >= 500 {
			level = slog.LevelError
		}
		slog.Log(r.Context(), level, "change balance failed", "error", err, "operation", operationType)
		writeJSON(w, statusCode, response)
		return
	}

	var response ApplyBalanceResponse
	if operationType == domain.OperationTypeDeposit {
		response.Message = MessageApplyBalanceSuccessDeposit
	} else {
		response.Message = MessageApplyBalanceSuccessWithdraw
	}
	response.NewBalance = make(map[string]json.Number)
	for _, v := range balances {
		amount := v.Amount()
		response.NewBalance[string(v.Currency().CurrencyType())] = amountInt64ToJsonNumber(amount)
	}

	slog.InfoContext(r.Context(), "balance changed", "user_id", userID.String(), "operation", operationType, "amount", amount)
	writeJSON(w, http.StatusOK, response)
}

func jsonNumberToAmountInt64(input json.Number) (int64, error) {
	if strings.ContainsAny(input.String(), "eE") {
		return 0, fmt.Errorf("%w: %+v", errInvalidInputAmount, input)
	}

	parts := strings.Split(input.String(), ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("%w: %+v", errInvalidInputAmount, input)
	}
	if strings.TrimSpace(parts[0]) == "" {
		return 0, fmt.Errorf("%w: %+v", errInvalidInputAmount, input)
	}
	if len(parts) == 2 && len(parts[1]) > 2 {
		return 0, fmt.Errorf("%w: %+v", errInvalidInputAmount, input)
	}
	if len(parts) == 2 {
		if len(parts[1]) == 1 {
			parts[1] = parts[1] + "0"
		} else if len(parts[1]) == 0 {
			parts[1] = parts[1] + "00"
		}
	}
	if len(parts) == 1 {
		parts = append(parts, "00")
	}
	amount, err := strconv.ParseInt(strings.Join(parts, ""), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("convert json number to int 64: %w: %+v", errors.Join(err, errInvalidInputAmount), input)
	}
	return amount, nil
}

func amountInt64ToJsonNumber(amount int64) json.Number {
	return json.Number(
		fmt.Sprintf("%d.%02d", amount/100, amount%100),
	)
}
