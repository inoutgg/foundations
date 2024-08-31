package token

import (
	"errors"
	"strings"
)

var (
	ErrMalformedToken = errors.New("foundations/token: invalid format")
)

// TokenFromBearerString returns the token from a bearer token string.
func TokenFromBearerString(str string) (string, error) {
	if !strings.HasPrefix(str, "Bearer ") {
		return "", ErrMalformedToken
	}

	tok := strings.TrimSpace(strings.TrimPrefix(str, "Bearer "))
	if tok == "" {
		return "", ErrMalformedToken
	}

	return tok, nil
}
