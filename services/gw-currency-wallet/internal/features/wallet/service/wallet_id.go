package service

import (
	"wallet-app/internal/core/domain"

	"github.com/google/uuid"
)

func GenerateWalletID() (domain.WalletID, error) {
	value := uuid.New()
	walletID, err := domain.NewWalletID(value)
	if err != nil {
		return domain.WalletID{}, err
	}
	return walletID, nil
}
