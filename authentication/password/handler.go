package password

import (
	"context"
	"errors"
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
	"go.inout.gg/common/pointer"
	"go.inout.gg/common/sql/dbutil"
	"go.inout.gg/common/uuidv7"
)

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
	HijackUserLogin(context.Context, uuid.UUID) (T, error)
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
	usr := user.FromContext[any](ctx)
	if usr != nil {
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

func (h *Handler[T]) HandleUserLogin(
	ctx context.Context,
	email, password string,
) (*strategy.User[T], error) {
	// Forbid authorized user access.
	usr := user.FromContext[any](ctx)
	if usr != nil {
		return nil, authentication.ErrAuthorizedUser
	}

	q := h.driver.Queries()

	user, err := q.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, authentication.ErrUserNotFound
		}

		return nil, fmt.Errorf("authentication/password: failed to find user: %w", err)
	}

	passwordHash := pointer.ToValue(user.PasswordHash, "")
	if passwordHash == "" {
		return nil, authentication.ErrUserNotFound
	}

	ok, err := h.config.PasswordHasher.Verify(passwordHash, password)
	if err != nil {
		return nil, fmt.Errorf("authentication/password: failed to validate password: %w", err)
	}

	if !ok {
		return nil, ErrPasswordIncorrect
	}

	// An entry point for hijacking the user registration process.
	var payload T
	if h.config.Hijacker != nil {
		payload, err = h.config.Hijacker.HijackUserLogin(ctx, user.ID)
		if err != nil {
			return nil, fmt.Errorf(
				"authentication/password: failed to hijack user login: %w",
				err,
			)
		}
	}

	return &strategy.User[T]{
		ID: user.ID,
		T:  payload,
	}, nil
}
