package union

import (
	"context"
	"errors"
	"net/http"

	"go.inout.gg/common/authentication/strategy"
	"go.inout.gg/common/authentication/token"
)

var _ strategy.Authenticator[any] = (*tok[any])(nil)

var (
	ErrInvalidToken = errors.New("authentication/token: invalid token")
)

type tok[T any] struct{}

func New[T any]() strategy.Authenticator[T] {
	return &tok[T]{}
}

func (t tok[T]) Authenticate(
	ctx context.Context,
	r *http.Request,
) (*strategy.User[T], error) {
	token, err := token.FromRequest(r)
	if err != nil {
		return nil, err
	}

	print(token)

	return nil, ErrInvalidToken
}
