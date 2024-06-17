// uuidv7 is a wrapper around google's uuid package.
package uuidv7

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.inout.gg/common/must"
)

// Must returns a new random UUID. It panics if there is an error.
func Must() uuid.UUID {
	return must.Must(uuid.NewV7())
}

// FromString parses a UUID from a string.
func FromString(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ToPgxUUID converts a UUID to a pgx.UUID.
func ToPgxUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// FromPgxUUID converts a pgx.UUID to a UUID.
func FromPgxUUID(u pgtype.UUID) (uuid.UUID, error) {
	return uuid.FromBytes(u.Bytes[:])
}

func MustFromPgxUUID(u pgtype.UUID) uuid.UUID {
	return must.Must(FromPgxUUID(u))
}
