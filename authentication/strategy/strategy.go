package strategy

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type User[T any] struct {
	ID uuid.UUID
	T  T
}

// Authenticator authenticates the user.
type Authenticator[T any] interface {
	Authenticate(ctx context.Context, r *http.Request) (*User[T], error)
}
