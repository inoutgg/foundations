package password

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.inout.gg/common/authentication"
	"go.inout.gg/common/authentication/db/driver"
	"go.inout.gg/common/authentication/internal/query"
	"go.inout.gg/common/authentication/password/verification"
	"go.inout.gg/common/authentication/strategy"
	"go.inout.gg/common/authentication/user"
	"go.inout.gg/common/debug"
	"go.inout.gg/common/internal/uuidv7"
	"go.inout.gg/common/sql/db/dbutil"
)

var d = debug.Debuglog("authentication/password")

var (
	ErrEmailAlreadyTaken = fmt.Errorf("authentication/password: email already taken")
	ErrPasswordIncorrect = fmt.Errorf("authentication/password: password incorrect")
)

// Config is the configuration for the password handler.
type Config[T any] struct {
	Logger         *slog.Logger
	PasswordHasher PasswordHasher
	Hijacker       Hijacker[T]
}

type Hijacker[T any] interface {
	HijackUserRegisteration(context.Context, uuid.UUID, pgx.Tx) (T, error)

	// HijackUserLogin is called when a user is trying to login.
	// Use this method to fetch additional data from the database for the user.
	//
	// Note that the user password is not verified at this moment yet.
	HijackUserLogin(context.Context, uuid.UUID, pgx.Tx) (T, error)
}

// NewConfig creates a new config.
//
// If no password hasher is configured, the DefaultPasswordHasher will be used.
func NewConfig[T any](config ...func(*Config[T])) *Config[T] {
	cfg := Config[T]{}

	for _, c := range config {
		c(&cfg)
	}

	if cfg.PasswordHasher == nil {
		cfg.PasswordHasher = DefaultPasswordHasher
	}

	debug.Assert(cfg.PasswordHasher != nil, "PasswordHasher must be set")
	debug.Assert(cfg.Logger != nil, "Logger must be set")

	return &cfg
}

// WithPasswordHasher configures the password hasher.
//
// When setting a password hasher make sure to set it across all modules,
// such as user registration, password reset and password verification.
func WithPasswordHasher[T any](hasher PasswordHasher) func(*Config[T]) {
	return func(cfg *Config[T]) { cfg.PasswordHasher = hasher }
}

func WithLogger[T any](logger *slog.Logger) func(*Config[T]) {
	return func(cfg *Config[T]) { cfg.Logger = logger }
}

func WithHijacker[T any](hijacker Hijacker[T]) func(*Config[T]) {
	return func(cfg *Config[T]) { cfg.Hijacker = hijacker }
}

type Handler[T any] struct {
	config           *Config[T]
	driver           driver.Driver
	PasswordVerifier verification.PasswordVerifier
}

func (h *Handler[T]) HandleUserRegistration(
	ctx context.Context,
	email, password string,
) (*strategy.User[T], error) {
	// Forbid authorized user access.
	if user.IsAuthorized(ctx) {
		return nil, authentication.ErrAuthorizedUser
	}

	// Make sure that the password hashing is performed outside of the transaction
	// as it is an expensive operation.
	passwordHash, err := h.config.PasswordHasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("authentication/password: failed to hash password: %w", err)
	}

	tx, err := h.driver.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication/password: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	uid, err := h.handleUserRegistrationTx(ctx, email, passwordHash, tx)
	if err != nil {
		return nil, err
	}

	// An entry point for hijacking the user registration process.
	var payload T
	if h.config.Hijacker != nil {
		d("registration hijacking is enabled, trying to get payload")
		payload, err = h.config.Hijacker.HijackUserRegisteration(ctx, uid, tx.Tx())
		if err != nil {
			return nil, fmt.Errorf(
				"authentication/password: failed to hijack user registration: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("authentication/password: failed to register a user: %w", err)
	}

	return &strategy.User[T]{
		ID: uid,
		T:  payload,
	}, nil
}

func (h *Handler[T]) handleUserRegistrationTx(
	ctx context.Context,
	email, passwordHash string,
	tx driver.ExecutorTx,
) (uuid.UUID, error) {
	uid := uuidv7.Must()
	q := tx.Queries()
	if err := q.CreateUser(ctx, query.CreateUserParams{
		ID:    uuidv7.ToPgxUUID(uid),
		Email: email,
	}); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return uid, ErrEmailAlreadyTaken
		}

		return uid, fmt.Errorf("authentication/password: failed to register a user: %w", err)
	}

	if err := q.CreateUserPasswordCredential(ctx, query.CreateUserPasswordCredentialParams{
		ID:                   uuidv7.ToPgxUUID(uuidv7.Must()),
		UserID:               uuidv7.ToPgxUUID(uid),
		UserCredentialKey:    email,
		UserCredentialSecret: passwordHash,
	}); err != nil {
		return uid, fmt.Errorf("authentication/password: failed to register a user: %w", err)
	}

	return uid, nil
}

func (h *Handler[T]) HandleUserLogin(
	ctx context.Context,
	email, password string,
) (*strategy.User[T], error) {
	// Forbid authorized user access.
	if user.IsAuthorized(ctx) {
		return nil, authentication.ErrAuthorizedUser
	}

	tx, err := h.driver.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication/password: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := tx.Queries()
	user, err := q.FindUserWithPasswordCredentialByEmail(
		ctx,
		email,
	)
	if err != nil {
		if dbutil.IsNotFoundError(err) {
			return nil, authentication.ErrUserNotFound
		}

		return nil, fmt.Errorf("authentication/password: failed to find user: %w", err)
	}

	// Treat the empty password as a non-existing user/credential.
	if user.PasswordHash == "" {
		d("empty password in db")
		return nil, authentication.ErrUserNotFound
	}

	uid := uuidv7.MustFromPgxUUID(user.ID)

	// An entry point for hijacking the user login process.
	var payload T
	if h.config.Hijacker != nil {
		d("login hijacking is enabled, trying to get payload")
		payload, err = h.config.Hijacker.HijackUserLogin(ctx, uid, tx.Tx())
		if err != nil {
			return nil, fmt.Errorf(
				"authentication/password: failed to hijack user login: %w",
				err,
			)
		}
	}

	// Make sure that the password hashing is performed outside of the transaction.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("authentication/password: failed to login a user: %w", err)
	}

	ok, err := h.config.PasswordHasher.Verify(user.PasswordHash, password)
	if err != nil {
		return nil, fmt.Errorf("authentication/password: failed to verify password: %w", err)
	}

	if !ok {
		d("password mismatch")
		return nil, ErrPasswordIncorrect
	}

	return &strategy.User[T]{
		ID: uid,
		T:  payload,
	}, nil
}
