package password

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"
)

var _ PasswordHasher = (*bcryptPasswordHasher)(nil)

const (
	BcryptDefaultCost = bcrypt.DefaultCost
)

type bcryptPasswordHasher struct {
	cost int
}

// NewBcryptPasswordHasher implements a password hashing algorithm with bcrypt.
func NewBcryptPasswordHasher(cost int) PasswordHasher {
	return &bcryptPasswordHasher{cost}
}

func (h *bcryptPasswordHasher) Hash(password string) (string, error) {
	passwordBytes := []byte(norm.NFKC.String(password))
	hash, err := bcrypt.GenerateFromPassword(passwordBytes, h.cost)
	if err != nil {
		return "", fmt.Errorf("authentication/password: unable to generate a bcrypt hash: %w", err)
	}

	return string(hash), nil
}

func (h *bcryptPasswordHasher) Verify(hashedPassword string, password string) (bool, error) {
	passwordBytes := []byte(norm.NFKC.String(password))
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), passwordBytes); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}

		return false, fmt.Errorf(
			"authentication/password: failed while comparing passwords: %w",
			err,
		)
	}

	return true, nil
}
