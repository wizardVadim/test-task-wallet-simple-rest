package hash

import "golang.org/x/crypto/bcrypt"

type BcryptHasher struct {
}

func (h *BcryptHasher) Hash(from string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(from), bcrypt.DefaultCost)
	return string(bytes), err
}

func (h *BcryptHasher) Compare(from string, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(from))
}
