package authentication

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrAuthorizedUser   = errors.New("authentication: authorized user access")
	ErrUnauthorizedUser = errors.New("authentication: unauthorized user access")
	ErrUserNotFound     = errors.New("authentication: user not found")
)

type User[T any] struct {
	// ID is the user ID.
	ID uuid.UUID

	// T holds additional data.
	//
	// Make sure that the data is JSON-serializable.
	T *T
}
