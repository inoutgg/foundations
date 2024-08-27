package union

import (
	"errors"
	"net/http"

	"go.inout.gg/foundations/authentication/strategy"
)

var _ strategy.Authenticator[any] = (unionStrategy[any])(nil)

type unionStrategy[T any] []strategy.Authenticator[T]

// New creates a new Authenticator that tries multiple authenticators.
func New[T any](authenticators ...strategy.Authenticator[T]) strategy.Authenticator[T] {
	return unionStrategy[T](authenticators)
}

func (u unionStrategy[T]) Authenticate(
	w http.ResponseWriter,
	r *http.Request,
) (*strategy.Session[T], error) {
	errs := make([]error, 0)

	for _, authenticator := range u {
		user, err := authenticator.Authenticate(w, r)
		if err != nil {
			errs = append(errs, err)
		} else {
			return user, nil
		}
	}

	return nil, errors.Join(errs...)
}

func (_ unionStrategy[T]) Issue(
	http.ResponseWriter,
	*http.Request,
	*strategy.User[T],
) (*strategy.Session[T], error) {
	return nil, errors.ErrUnsupported
}
