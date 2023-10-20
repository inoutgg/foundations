package passwordless

import "time"

type Token[State comparable] interface {
	// GetUserID returns the user unique identifier associated with the token
	// (e.g. an email address or a username).
	UserID() string

	// CreatedAt returns the time when the token was created.
	CreatedAt() time.Time

	// ExpiresAt returns the time when the token will expire.
	ExpiresAt() time.Time

	// State returns an additional information that might be carrier by the token.
	State() State
}

type TokenStore[TokenState comparable] interface {
	Delete(Token[TokenState]) error
	Find(string) (Token[TokenState], error)
}

type Passwordless[TokenState comparable] struct {
	store TokenStore[TokenState]
}

func New[TokenState comparable](store TokenStore[TokenState]) Passwordless[TokenState] {
	return Passwordless[TokenState]{store}
}
