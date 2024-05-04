package union

import (
	"context"
	"errors"
	"net/http"

	"go.inout.gg/common/authentication/strategy"
)

var _ strategy.Authenticator[any] = (union[any])(nil)

type union[T any] []strategy.Authenticator[T]

// New creates a new Authenticator that tries multiple authenticators.
func New[T any](authenticators ...strategy.Authenticator[T]) strategy.Authenticator[T] {
	return union[T](authenticators)
}

func (union union[T]) Authenticate(
	ctx context.Context,
	r *http.Request,
) (*strategy.User[T], error) {
	errs := make([]error, 0)

	for _, authenticator := range union {
		user, err := authenticator.Authenticate(ctx, r)
		if err != nil {
			errs = append(errs, err)
		} else {
			return user, nil
		}
	}

	return nil, errors.Join(errs...)
}
