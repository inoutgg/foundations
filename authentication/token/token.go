package token

import (
	"errors"
	"net/http"

	httperror "go.inout.gg/common/http/error"
	"go.inout.gg/common/token"
)

var (
	ErrTokenNotFound = errors.New("authentication/token: no token found")
)

// FromRequest returns the token from the given HTTP request.
func FromRequest(req *http.Request) (string, error) {
	value := req.Header.Get("Authorization")
	if value != "" {
		tok, err := token.TokenFromBearerString(value)
		if err != nil {
			return "", httperror.FromError(err, http.StatusUnauthorized)
		}

		return tok, nil
	}

	return "", httperror.FromError(ErrTokenNotFound, http.StatusUnauthorized)
}
