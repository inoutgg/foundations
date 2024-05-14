package password

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/common/authentication/internal/query"
	"go.inout.gg/common/authentication/password/verification"
	"go.inout.gg/common/pointer"
	"go.inout.gg/common/sql/dbutil"
	"go.inout.gg/common/uuidv7"
)

var (
	ErrEmailAlreadyTaken = fmt.Errorf("authentication/password: email already taken")
	ErrUserNotFound      = fmt.Errorf("authentication/password: user not found")
	ErrPasswordIncorrect = fmt.Errorf("authentication/password: password incorrect")
)

type EmailAndPasswordProvider struct {
	PasswordHasher   PasswordHasher
	PasswordVerifier verification.PasswordVerifier

	logger slog.Logger
	query.Queries
	pool pgxpool.Pool
}

func (p *EmailAndPasswordProvider) Register(
	ctx context.Context,
	email, password string,
) (uuid.UUID, error) {
	var uid uuid.UUID
	passwordHash, err := p.PasswordHasher.Hash(password)
	if err != nil {
		return uid, fmt.Errorf("authentication/password: failed to hash password: %w", err)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return uid, fmt.Errorf("authentication/password: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	uid, err = p.registerTx(ctx, email, passwordHash, tx)
	if err != nil {
		return uid, err
	}

	if err := tx.Commit(ctx); err != nil {
		return uid, fmt.Errorf("authentication/password: failed to register a user: %w", err)
	}

	return uid, nil
}

func (p *EmailAndPasswordProvider) registerTx(
	ctx context.Context,
	email, passwordHash string,
	tx pgx.Tx,
) (uuid.UUID, error) {
	uid := uuidv7.Must()
	q := p.Queries.WithTx(tx)
	if err := q.CreateUser(ctx, query.CreateUserParams{
		ID:           uid,
		Email:        email,
		PasswordHash: pointer.FromValue(passwordHash),
	}); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return uid, ErrEmailAlreadyTaken
		}

		return uid, fmt.Errorf("authentication/password: failed to register a user: %w", err)
	}

	return uid, nil
}

func (p *EmailAndPasswordProvider) Authenticate(
	ctx context.Context,
	email, password string,
) (any, error) {
	user, err := p.Queries.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("authentication/password: failed to find user: %w", err)
	}

	passwordHash := pointer.ToValue(user.PasswordHash, "")
	if passwordHash == "" {
		return nil, ErrUserNotFound
	}

	ok, err := p.PasswordHasher.Verify(passwordHash, password)
	if err != nil {
		return nil, fmt.Errorf("authentication/password: failed to validate password: %w", err)
	}

	if !ok {
		return nil, ErrPasswordIncorrect
	}

	return user, nil
}
