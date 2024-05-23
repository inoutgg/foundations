package authentication

import (
	"errors"
)

var (
	ErrAuthorizedUser   = errors.New("authentication: authorized user access")
	ErrUnauthorizedUser = errors.New("authentication: unauthorized user access")
	ErrUserNotFound     = errors.New("authentication: user not found")
)
