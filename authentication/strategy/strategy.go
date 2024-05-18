package strategy

import (
	"net/http"

	"github.com/google/uuid"
)

type User[T any] struct {
	ID uuid.UUID
	T  T
}

// Authenticator authenticates the user.
type Authenticator[T any] interface {
	Authenticate(http.ResponseWriter, *http.Request) (*User[T], error)
}
