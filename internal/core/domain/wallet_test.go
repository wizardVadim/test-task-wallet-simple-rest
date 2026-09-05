package domain_test

import (
	"errors"
	"testing"
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

func TestNewWalletID(t *testing.T) {
	validUUID := uuid.New()
	invalidUUID := uuid.Nil
	tests := []struct {
		name     string
		input    uuid.UUID
		wantUUID uuid.UUID
		wantErr  error
	}{
		{name: "Simple init", input: validUUID, wantUUID: validUUID, wantErr: nil},
		{name: "Invalid UUID", input: invalidUUID, wantUUID: invalidUUID, wantErr: domain.ErrWalletValueIsEmpty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			walletID, err := domain.NewWalletID(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewWalletID() error = %v; want %v", err, tt.wantErr)
			}

			if walletID.Value() != tt.wantUUID {
				t.Fatalf("NewWalletID() not expected wallet id value; got: %v; want: %v", walletID.Value(), tt.wantUUID)
			}
		})
	}

}

func mustWalletID(t *testing.T) domain.WalletID {
	t.Helper()

	id, err := domain.NewWalletID(uuid.New())
	if err != nil {
		t.Fatalf("NewWalletID(): %v", err)
	}
	return id
}

func TestNewWallet(t *testing.T) {
	walletID := mustWalletID(t)
	tests := []struct {
		name         string
		inputID      domain.WalletID
		inputBalance int64
		wantID       domain.WalletID
		wantBalance  int64
		wantErr      error
	}{
		{name: "Simple init", inputID: walletID, inputBalance: 100, wantID: walletID, wantBalance: 100, wantErr: nil},
		{name: "Zero balance", inputID: walletID, inputBalance: 0, wantID: walletID, wantBalance: 0, wantErr: nil},
		{name: "Empty wallet ID", inputID: domain.WalletID{}, inputBalance: 100, wantID: domain.WalletID{}, wantBalance: 0, wantErr: domain.ErrWalletValueIsEmpty},
		{name: "Invalid balance", inputID: walletID, inputBalance: -1000, wantID: domain.WalletID{}, wantBalance: 0, wantErr: domain.ErrBalanceMustBePositiveOrZero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			wallet, err := domain.NewWallet(tt.inputID, tt.inputBalance)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewWallet() error = %v; want %v", err, tt.wantErr)
			}

			if !wallet.ID().IsEqual(tt.wantID) {
				t.Fatalf("NewWallet() not expected wallet id; got: %v; want: %v", wallet.ID(), tt.wantID)
			}

			if wallet.Balance() != tt.wantBalance {
				t.Fatalf("NewWallet() not expected balance; got: %v; want: %v", wallet.Balance(), tt.wantBalance)
			}
		})
	}

}

func TestNewWalletOperation(t *testing.T) {
	walletID := mustWalletID(t)
	tests := []struct {
		name               string
		inputID            domain.WalletID
		inputAmount        int64
		inputOperationType domain.OperationType
		wantID             domain.WalletID
		wantAmount         int64
		wantOperationType  domain.OperationType
		wantErr            error
	}{
		{
			name:               "Simple init deposit",
			inputID:            walletID,
			inputAmount:        100,
			inputOperationType: domain.OperationTypeDeposit,
			wantID:             walletID,
			wantAmount:         100,
			wantOperationType:  domain.OperationTypeDeposit,
			wantErr:            nil,
		},
		{
			name:               "Simple init withdraw",
			inputID:            walletID,
			inputAmount:        100,
			inputOperationType: domain.OperationTypeWithdraw,
			wantID:             walletID,
			wantAmount:         100,
			wantOperationType:  domain.OperationTypeWithdraw,
			wantErr:            nil,
		},
		{
			name:               "Invalid operation type",
			inputID:            walletID,
			inputAmount:        100,
			inputOperationType: domain.OperationType("invalid"),
			wantID:             domain.WalletID{},
			wantAmount:         0,
			wantOperationType:  domain.OperationType(""),
			wantErr:            domain.ErrInvalidOperationType,
		},
		{
			name:               "Invalid amount",
			inputID:            walletID,
			inputAmount:        -1,
			inputOperationType: domain.OperationTypeDeposit,
			wantID:             domain.WalletID{},
			wantAmount:         0,
			wantOperationType:  domain.OperationType(""),
			wantErr:            domain.ErrAmountMustBePositive,
		},
		{
			name:               "Invalid amount zero",
			inputID:            walletID,
			inputAmount:        0,
			inputOperationType: domain.OperationTypeDeposit,
			wantID:             domain.WalletID{},
			wantAmount:         0,
			wantOperationType:  domain.OperationType(""),
			wantErr:            domain.ErrAmountMustBePositive,
		},
		{
			name:               "Empty wallet ID",
			inputID:            domain.WalletID{},
			inputAmount:        100,
			inputOperationType: domain.OperationTypeDeposit,
			wantID:             domain.WalletID{},
			wantAmount:         0,
			wantOperationType:  domain.OperationType(""),
			wantErr:            domain.ErrWalletValueIsEmpty,
		},
		{
			name:               "Init deposit UP",
			inputID:            walletID,
			inputAmount:        100,
			inputOperationType: domain.OperationType("DEPOSIT"),
			wantID:             walletID,
			wantAmount:         100,
			wantOperationType:  domain.OperationTypeDeposit,
			wantErr:            nil,
		},
		{
			name:               "Init withdraw UP",
			inputID:            walletID,
			inputAmount:        100,
			inputOperationType: domain.OperationType("WITHDRAW"),
			wantID:             walletID,
			wantAmount:         100,
			wantOperationType:  domain.OperationTypeWithdraw,
			wantErr:            nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			operation, err := domain.NewWalletOperation(tt.inputID, tt.inputOperationType, tt.inputAmount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewWalletOperation() error = %v; want %v", err, tt.wantErr)
			}

			if !operation.WalletID().IsEqual(tt.wantID) {
				t.Fatalf("NewWalletOperation() not expected wallet id; got: %v; want: %v", operation.WalletID(), tt.wantID)
			}

			if operation.Amount() != tt.wantAmount {
				t.Fatalf("NewWalletOperation() not expected amount; got: %v; want: %v", operation.Amount(), tt.wantAmount)
			}

			if operation.OperationType() != tt.wantOperationType {
				t.Fatalf("NewWalletOperation() not expected opearation type; got: %v; want: %v", operation.OperationType(), tt.wantOperationType)
			}
		})
	}

}
