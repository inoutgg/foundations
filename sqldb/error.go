package sqldb

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/puddle/v2"
)

const (
	ErrCodeUniqueViolation = "23505"
)

// IsUniqueViolationError returns true if the error is a unique violation error.
func IsUniqueViolationError(err error) bool {
	pgxErr := &pgconn.PgError{}
	if errors.As(err, &pgxErr) {
		return pgxErr.Code == ErrCodeUniqueViolation
	}

	return false
}

// IsNotFoundError returns true if the error is a pgx no rows error.
func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// IsPoolClosed returns true if the error is a pgx closed pool error.
func IsPoolClosed(err error) bool {
	return errors.Is(err, puddle.ErrClosedPool)
}
