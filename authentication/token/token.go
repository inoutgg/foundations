package token

import (
	"net/http"

	httperror "github.com/atcirclesquare/common/http/error"
	"github.com/atcirclesquare/common/token"
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

	return "", httperror.New(
		"authentication/token: no token found, \"AUTHORIZATION\" header is missing",
		http.StatusUnauthorized,
	)
}
