package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

type User struct {
	id           uuid.UUID
	username     string
	email        string
	passwordHash string
}

func NewUser(id uuid.UUID, username string, email string, passwordHash string) (User, error) {
	emailAddress, err := mail.ParseAddress(email)
	if err != nil {
		return User{}, errors.Join(err, ErrInvalidEmailAddress)
	}

	user := User{
		id:           id,
		username:     strings.ToLower(strings.TrimSpace(username)),
		email:        emailAddress.Address,
		passwordHash: passwordHash,
	}

	if err := user.validate(); err != nil {
		return User{}, err
	}
	return user, nil
}

func (u User) validate() error {
	if u.id == uuid.Nil {
		return fmt.Errorf("%w: %s", ErrInvalidUserID, "nil")
	}
	if u.username == "" {
		return fmt.Errorf("%w: %s", ErrInvalidUsername, "empty")
	}
	if u.email == "" {
		return fmt.Errorf("%w: %s", ErrInvalidEmailAddress, "empty")
	}
	if u.passwordHash == "" {
		return fmt.Errorf("%w: %s", ErrInvalidPasswordHash, "empty")
	}

	return nil
}

func (u User) ID() uuid.UUID {
	return u.id
}

func (u User) Username() string {
	return u.username
}

func (u User) Email() string {
	return u.email
}

func (u User) PasswordHash() string {
	return u.passwordHash
}
