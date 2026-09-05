package wallet_http

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"wallet-app/internal/core/domain"
	"wallet-app/internal/features/wallet/service"

	"github.com/google/uuid"
)

type Handler struct {
	walletService WalletService
}

func New(walletService WalletService) *Handler {
	return &Handler{walletService: walletService}
}

// GET /api/v1/wallets{wallet_uuid}
func (h *Handler) GetWalletBalance(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}

	parsedID, err := uuid.Parse(r.PathValue("wallet_uuid"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ResponseDTO{
			Error: &ErrorDTO{Message: ErrorInvalidWalletID},
		})
		return
	}

	walletID, err := domain.NewWalletID(parsedID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ResponseDTO{
			Error: &ErrorDTO{Message: ErrorInvalidWalletID},
		})
		return
	}

	balance, err := h.walletService.GetWalletBalance(r.Context(), walletID)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		if errors.Is(err, domain.ErrWalletNotFound) {
			writeJSON(w, http.StatusNotFound, ResponseDTO{
				Error: &ErrorDTO{Message: ErrorWalletNotFound},
			})
			return
		}

		log.Printf("get wallet balance: %v", err)
		writeJSON(w, http.StatusInternalServerError, ResponseDTO{
			Error: &ErrorDTO{Message: ErrorGetBalanceInternal},
		})
		return
	}

	writeJSON(w, http.StatusOK, ResponseDTO{
		Payload: GetBalancePayload{Balance: balance},
	})
}

// POST /api/v1/wallets
func (h *Handler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}

	wallet, err := h.walletService.CreateNewWallet(r.Context())
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		log.Printf("create wallet: %v", err)
		writeJSON(w, http.StatusInternalServerError, ResponseDTO{
			Error: &ErrorDTO{Message: ErrorCreateWalletInternal},
		})
		return
	}

	walletID := wallet.ID().Value().String()
	w.Header().Set("Location", "/api/v1/wallets/"+walletID)
	writeJSON(w, http.StatusCreated, ResponseDTO{
		Payload: CreateWalletPayload{
			WalletID: walletID,
			Balance:  wallet.Balance(),
		},
	})
}

// POST /api/v1/wallet
//
// request body:
//
//	{
//		"walletID": "9c2d217e-96d3-4117-9f0b-3952c4dc6ec2",
//		"operationType": "DEPOSIT",
//		"amount": 1000
//	}
func (h *Handler) ChangeWalletBalance(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}

	var changeBalanceDTO ChangeBalanceDTO

	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&changeBalanceDTO)
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

		status := http.StatusBadRequest
		message := ErrorInvalidRequestBody

		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			status = http.StatusRequestEntityTooLarge
			message = ErrorRequestBodyTooLarge
		}

		writeJSON(w, status, ResponseDTO{
			Error: &ErrorDTO{Message: message},
		})
		return
	}

	parsedID, err := uuid.Parse(changeBalanceDTO.WalletID)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		log.Printf("change wallet balance: %v", err)
		writeJSON(w, http.StatusBadRequest, ResponseDTO{
			Error: &ErrorDTO{Message: ErrorInvalidRequestBody},
		})
		return
	}

	walletID, err := domain.NewWalletID(parsedID)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		log.Printf("change wallet balance: %v", err)
		writeJSON(w, http.StatusBadRequest, ResponseDTO{
			Error: &ErrorDTO{Message: ErrorInvalidWalletID},
		})
		return
	}

	operation, err := domain.NewWalletOperation(
		walletID,
		domain.OperationType(changeBalanceDTO.OperationType),
		changeBalanceDTO.Amount,
	)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}

		log.Printf("change wallet balance: %v", err)
		writeJSON(w, http.StatusBadRequest, ResponseDTO{
			Error: &ErrorDTO{Message: ErrorInvalidRequestBody},
		})
		return
	}

	if err := h.walletService.ChangeWalletBalance(r.Context(), operation); err != nil {
		if r.Context().Err() != nil {
			return
		}

		var response ResponseDTO
		var statusCode int

		if errors.Is(err, service.ErrSmallBalance) {
			response = ResponseDTO{
				Error: &ErrorDTO{
					Message: ErrorInsufficientFunds,
				},
			}
			statusCode = http.StatusConflict
		} else if errors.Is(err, service.ErrBalanceOverflow) {
			response = ResponseDTO{
				Error: &ErrorDTO{
					Message: ErrorBalanceOverflow,
				},
			}
			statusCode = http.StatusUnprocessableEntity
		} else if errors.Is(err, domain.ErrWalletNotFound) {
			response = ResponseDTO{
				Error: &ErrorDTO{
					Message: ErrorWalletNotFound,
				},
			}
			statusCode = http.StatusNotFound
		} else {
			response = ResponseDTO{
				Error: &ErrorDTO{
					Message: ErrorChangeBalanceInternal,
				},
			}
			statusCode = http.StatusInternalServerError
		}

		log.Printf("change wallet balance: %v", err)
		writeJSON(w, statusCode, response)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, response ResponseDTO) {
	body, err := json.Marshal(response)
	if err != nil {
		log.Printf("marshal response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}
