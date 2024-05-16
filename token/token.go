package token

import (
	"errors"
	"strings"
)

var (
	ErrMalformedToken = errors.New("token: invalid format")
)

// TokenFromBearerString returns the token from a bearer token string.
func TokenFromBearerString(str string) (string, error) {
	if !strings.HasPrefix(str, "Bearer ") {
		return "", ErrMalformedToken
	}

	return strings.TrimPrefix(strings.TrimSpace(str), "Bearer "), nil
}
