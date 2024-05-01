package password

import (
	"context"
	"errors"
	"fmt"

	"github.com/atcirclesquare/common/authentication/internal/query"
	"github.com/atcirclesquare/common/authentication/password/verification"
	"github.com/atcirclesquare/common/authentication/routes"
	"github.com/atcirclesquare/common/http/routerutil"
	"github.com/atcirclesquare/common/pointer"
	"github.com/atcirclesquare/common/sql/dbutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ routes.Applicator = (*EmailAndPasswordProvider)(nil)

var (
	ErrEmailAlreadyTaken = fmt.Errorf("authentication/password: email already taken")
	ErrUserNotFound      = fmt.Errorf("authentication/password: user not found")
	ErrPasswordIncorrect = fmt.Errorf("authentication/password: password incorrect")
)

type EmailAndPasswordProvider struct {
	PasswordHasher   PasswordHasher
	PasswordVerifier verification.PasswordVerifier

	query.Queries
	pgxpool.Pool
}

func (p *EmailAndPasswordProvider) Routes(config *routes.Config) routerutil.Applicator {
	return &routesAplicator{config, p}
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

	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return uid, fmt.Errorf("authentication/password: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := p.Queries.WithTx(tx)
	uid, err = q.CreateUser(ctx, query.CreateUserParams{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: &passwordHash,
	})
	if err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return uid, ErrEmailAlreadyTaken
		}

		return uid, fmt.Errorf("authentication/password: failed to register a user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
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
