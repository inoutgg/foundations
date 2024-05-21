package password

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	ErrAuthorizedUser    = fmt.Errorf("authentication/password: authorized user access")
	ErrEmailAlreadyTaken = fmt.Errorf("authentication/password: email already taken")
	ErrUserNotFound      = fmt.Errorf("authentication/password: user not found")
	ErrPasswordIncorrect = fmt.Errorf("authentication/password: password incorrect")
)

// Config is the configuration for the password handler.
type Config struct {
	Logger         *slog.Logger
	PasswordHasher PasswordHasher
	Hijacker       Hijacker
}

// TODO: implement
type Hijacker interface {
	HijackUserRegisteration(ctx context.Context, tx pgx.Tx) error
	HijackUserLogin(ctx context.Context) error
}

// NewConfig creates a new config.
//
// If no password hasher is configured, the DefaultPasswordHasher will be used.
func NewConfig(config ...func(*Config)) *Config {
	cfg := Config{}

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
func WithPasswordHasher(hasher PasswordHasher) func(*Config) {
	return func(cfg *Config) { cfg.PasswordHasher = hasher }
}

func WithLogger(logger *slog.Logger) func(*Config) {
	return func(cfg *Config) { cfg.Logger = logger }
}

type Handler struct {
	config           *Config
	driver           driver.Driver
	PasswordVerifier verification.PasswordVerifier
}

func (h *Handler) HandleUserRegistration(
	ctx context.Context,
	email, password string,
) (uuid.UUID, error) {
	var uid uuid.UUID

	// Forbid authorized user access.
	usr := user.FromContext[any](ctx)
	if usr != nil {
		return uid, ErrAuthorizedUser
	}

	// Make sure that the password hashing is performed outside of the transaction
	// as it is an expensive operation.
	passwordHash, err := h.config.PasswordHasher.Hash(password)
	if err != nil {
		return uid, fmt.Errorf("authentication/password: failed to hash password: %w", err)
	}

	tx, err := h.driver.Begin(ctx)
	if err != nil {
		return uid, fmt.Errorf("authentication/password: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	uid, err = h.handleUserRegistrationTx(ctx, email, passwordHash, tx)
	if err != nil {
		return uid, err
	}

	// An entry point for hijacking the user registration process.
	if h.config.Hijacker != nil {
		if err := h.config.Hijacker.HijackUserRegisteration(ctx, tx.Tx()); err != nil {
			return uid, fmt.Errorf(
				"authentication/password: failed to hijack user registration: %w",
				err,
			)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return uid, fmt.Errorf("authentication/password: failed to register a user: %w", err)
	}

	return uid, nil
}

func (h *Handler) handleUserRegistrationTx(
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

func (p *Handler) HandleUserLogin(
	ctx context.Context,
	email, password string,
) (*strategy.User[any], error) {
	// Forbid authorized user access.
	usr := user.FromContext[any](ctx)
	if usr != nil {
		return nil, ErrAuthorizedUser
	}

	q := p.driver.Queries()

	user, err := q.FindUserByEmail(ctx, email)
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

	ok, err := p.config.PasswordHasher.Verify(passwordHash, password)
	if err != nil {
		return nil, fmt.Errorf("authentication/password: failed to validate password: %w", err)
	}

	if !ok {
		return nil, ErrPasswordIncorrect
	}

	return &strategy.User[any]{
		ID: user.ID,
	}, nil
}
