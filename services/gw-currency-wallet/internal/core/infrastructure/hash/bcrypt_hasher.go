package hash

import (
	"errors"
	"wallet-app/internal/core/domain"

	"golang.org/x/crypto/bcrypt"
)

const maxByteLength = 72

type BcryptHasher struct {
}

func New() *BcryptHasher {
	return &BcryptHasher{}
}

func (h *BcryptHasher) Hash(from string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(from), bcrypt.DefaultCost)
	return string(bytes), err
}

func (h *BcryptHasher) Compare(from string, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(from))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return domain.ErrPasswordMismatch
	}
	return err
}

func (h *BcryptHasher) MaxByteLength() int {
	return maxByteLength
}
