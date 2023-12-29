package token

import (
	"errors"
	"strings"
)

// TokenFromBearerString returns the token from a bearer token string.
func TokenFromBearerString(str string) (string, error) {
	if !strings.HasPrefix(str, "Bearer") {
		return "", errors.New("invalid bearer string")
	}

	return strings.TrimPrefix(strings.TrimSpace(str), "Bearer "), nil
}
