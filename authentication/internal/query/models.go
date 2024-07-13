// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package query

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PasswordResetToken struct {
	ID        uuid.UUID
	CreatedAt pgtype.Timestamp
	UpdatedAt pgtype.Timestamp
	IsUsed    bool
	Token     string
	ExpiresAt pgtype.Timestamp
	UserID    uuid.UUID
}

type User struct {
	ID              uuid.UUID
	CreatedAt       pgtype.Timestamp
	UpdatedAt       pgtype.Timestamp
	Email           string
	IsEmailVerified bool
	FirstName       *string
	LastName        *string
}

type UserCredential struct {
	ID                   uuid.UUID
	CreatedAt            pgtype.Timestamp
	UpdatedAt            pgtype.Timestamp
	Name                 string
	UserID               uuid.UUID
	UserCredentialKey    string
	UserCredentialSecret string
}

type UserEmailVerificationToken struct {
	ID        uuid.UUID
	CreatedAt pgtype.Timestamp
	UpdatedAt pgtype.Timestamp
	IsUsed    bool
	Token     string
	Email     string
	UserID    uuid.UUID
}

type UserSession struct {
	ID        uuid.UUID
	CreatedAt pgtype.Timestamp
	UpdatedAt pgtype.Timestamp
	ExpiresAt pgtype.Timestamp
	UserID    uuid.UUID
	EvictedBy uuid.UUID
}
