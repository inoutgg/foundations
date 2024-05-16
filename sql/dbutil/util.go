package dbutil

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	ErrCodeUniqueViolation = "23505"
)

// IsUniqueViolationError returns true if the error is a unique violation error.
func IsUniqueViolationError(err error) bool {
	if pgxErr, ok := err.(*pgconn.PgError); !ok || pgxErr.Code != ErrCodeUniqueViolation {
		return false
	}

	return true
}

// IsNotFoundError returns true if the error is a pgx no rows error.
func IsNotFoundError(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}

	return false
}
