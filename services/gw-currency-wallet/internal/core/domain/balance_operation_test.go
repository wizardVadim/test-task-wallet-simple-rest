package domain_test

import (
	"errors"
	"math"
	"testing"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

func TestNewBalanceOperation(t *testing.T) {
	userID := uuid.MustParse("e373039e-7060-4525-90bc-d5c7d7525a80")
	currency, err := domain.NewCurrency(domain.CurrencyTypeUSD)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name      string
		userID    uuid.UUID
		currency  domain.Currency
		operation domain.OperationType
		amount    int64
		wantErr   error
	}{
		{"deposit", userID, currency, domain.OperationTypeDeposit, 1, nil},
		{"withdraw", userID, currency, domain.OperationTypeWithdraw, 1, nil},
		{"maximum amount", userID, currency, domain.OperationTypeDeposit, math.MaxInt64, nil},
		{"nil user", uuid.Nil, currency, domain.OperationTypeDeposit, 1, domain.ErrInvalidUserID},
		{"zero currency", userID, domain.Currency{}, domain.OperationTypeDeposit, 1, domain.ErrInvalidCurrencyType},
		{"empty operation", userID, currency, "", 1, domain.ErrInvalidOperationType},
		{"unknown operation", userID, currency, "transfer", 1, domain.ErrInvalidOperationType},
		{"zero amount", userID, currency, domain.OperationTypeDeposit, 0, domain.ErrInvalidBalanceAmount},
		{"negative amount", userID, currency, domain.OperationTypeWithdraw, -1, domain.ErrInvalidBalanceAmount},
		{"minimum int64", userID, currency, domain.OperationTypeDeposit, math.MinInt64, domain.ErrInvalidBalanceAmount},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewBalanceOperation(tt.userID, tt.currency, tt.operation, tt.amount)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v; want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if got != (domain.BalanceOperation{}) {
					t.Error("invalid operation returned nonzero value")
				}
			} else {
				if got.UserID() != tt.userID {
					t.Errorf("UserID = %v; want %v", got.UserID(), tt.userID)
				}
				if got.Currency() != tt.currency {
					t.Errorf("Currency = %v; want %v", got.Currency(), tt.currency)
				}
				if got.OperationType() != tt.operation {
					t.Errorf("OperationType = %v; want %v", got.OperationType(), tt.operation)
				}
				if got.Amount() != tt.amount {
					t.Errorf("Amount = %d; want %d", got.Amount(), tt.amount)
				}
			}
		})
	}
}
