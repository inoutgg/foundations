package token

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.inout.gg/common/authentication/strategy"
	"go.inout.gg/common/authentication/token"
)

var _ strategy.Authenticator[any] = (*tok[any])(nil)

var (
	ErrInvalidToken = errors.New("authentication/token: invalid token")
)

type Storage[T any] interface {
	Retrieve(ctx context.Context, token string) (*strategy.User[T], error)
}

type tok[T any] struct {
	storage Storage[T]
}

// New returns a new authenticator that authenticates using a bearer token.
func New[T any](storage Storage[T]) strategy.Authenticator[T] {
	return &tok[T]{storage}
}

func (t *tok[T]) Authenticate(w http.ResponseWriter, r *http.Request) (*strategy.User[T], error) {
	ctx := r.Context()
	token, err := token.FromRequest(r)
	if err != nil {
		return nil, err
	}

	user, err := t.storage.Retrieve(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("authentication/token: failed to retrieve user: %w", err)
	}

	return user, nil
}
