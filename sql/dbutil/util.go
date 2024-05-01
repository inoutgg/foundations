package dbutil

import "github.com/jackc/pgx/v5/pgconn"

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
