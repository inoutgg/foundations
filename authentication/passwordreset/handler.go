package passwordreset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"go.inout.gg/common/authentication"
	"go.inout.gg/common/authentication/db/driver"
	"go.inout.gg/common/authentication/internal/query"
	"go.inout.gg/common/authentication/password"
	"go.inout.gg/common/authentication/sender"
	"go.inout.gg/common/authentication/user"
	"go.inout.gg/common/debug"
	"go.inout.gg/common/internal/uuidv7"
	"go.inout.gg/common/must"
	"go.inout.gg/common/random"
)

var (
	// ErrUsedPasswordResetToken is returned when the password reset token has already been used.
	ErrUsedPasswordResetToken = errors.New("password/reset: used password reset token")
)

const (
	ResetTokenExpiry = time.Duration(12 * time.Hour)
	ResetTokenLength = 32
)

// Config is the configuration for the password handler.
//
// Make sure to use the NewConfig function to create a new config, instead
// of instatiating the struct directly.
type Config struct {
	PasswordHasher password.PasswordHasher // optional
	Logger         *slog.Logger            // optional

	// TokenLength set the length of the reset token.
	//
	// Defaults to ResetTokenExpiry.
	TokenLength int // optinal
	// TokenExpiryIn set the expiry time of the reset token.
	TokenExpiryIn time.Duration // optional
}

// NewConfig creates a new config.
func NewConfig(config ...func(*Config)) *Config {
	cfg := Config{
		TokenExpiryIn: ResetTokenExpiry,
		TokenLength:   ResetTokenLength,
	}

	for _, f := range config {
		f(&cfg)
	}

	if cfg.PasswordHasher == nil {
		cfg.PasswordHasher = password.DefaultPasswordHasher
	}

	debug.Assert(cfg.PasswordHasher != nil, "password hasher should be set")

	return &cfg
}

// WithPasswordHasher configures the password hasher.
//
// When setting a password hasher make sure to set it across all modules,
// such as user registrration, password reset and password verification.
func WithPasswordHasher(hasher password.PasswordHasher) func(*Config) {
	return func(cfg *Config) { cfg.PasswordHasher = hasher }
}

// WithLogger configures the logger.
func WithLogger(logger *slog.Logger) func(*Config) {
	return func(cfg *Config) { cfg.Logger = logger }
}

// ResetTokenMessagePayload is the payload for the reset token message.
type PasswordResetRequestMessagePayload struct {
	Token string
}

// PasswordResetSuccessMessagePayload is the payload for the password reset success message.
type PasswordResetSuccessMessagePayload struct{}

// Handler handles password reset requests.
//
// It is a general enough implementation so it can be used for different
// communication methods.
//
// Check out the FormHandler for a ready to use implementation that handles
// HTTP form requests.
type Handler struct {
	config *Config
	driver driver.Driver
	sender sender.Sender
}

// HandlePasswordReset handles a password reset request.
func (h *Handler) HandlePasswordReset(
	ctx context.Context,
	email string,
) error {
	// Forbid authorized user access.
	if user.IsAuthorized(ctx) {
		return authentication.ErrAuthorizedUser
	}

	tx, err := h.driver.Begin(ctx)
	if err != nil {
		return fmt.Errorf("password/reset: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := tx.Queries()
	user, err := q.FindUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("password/reset: failed to find user: %w", err)
	}

	tokStr := must.Must(random.SecureHexString(h.config.TokenLength))
	tok, err := q.UpsertPasswordResetToken(ctx, query.UpsertPasswordResetTokenParams{
		ID:     uuidv7.Must(),
		Token:  tokStr,
		UserID: user.ID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(h.config.TokenExpiryIn),
			Valid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("password/reset: failed to upsert password reset token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("password/reset: failed to commit transaction: %w", err)
	}

	if err := h.sender.Send(ctx, sender.Message{
		Email: user.Email,
		Payload: PasswordResetRequestMessagePayload{
			Token: tok.Token,
		},
	}); err != nil {
		return fmt.Errorf("password/reset: failed to send password reset token: %w", err)
	}

	return nil
}

func (h *Handler) HandlePasswordResetConfirm(
	ctx context.Context,
	password, tokStr string,
) error {
	ph := h.config.PasswordHasher

	// NOTE: hash password upfront to avoid unnecessary database TX delay.
	passwordHash, err := ph.Hash(password)
	if err != nil {
		return fmt.Errorf("password/reset: failed to hash password: %w", err)
	}

	tx, err := h.driver.Begin(ctx)
	if err != nil {
		return fmt.Errorf("password/reset: failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	q := tx.Queries()
	tok, err := q.FindPasswordResetToken(ctx, tokStr)
	if err != nil {
		return fmt.Errorf("password/reset: failed to find password reset token: %w", err)
	}

	if tok.IsUsed {
		return ErrUsedPasswordResetToken
	}
	user, err := q.FindUserByID(ctx, tok.UserID)
	if err != nil {
		return fmt.Errorf("password/reset: failed to find user: %w", err)
	}

	if err := q.MarkPasswordResetTokenAsUsed(ctx, tok.Token); err != nil {
		return fmt.Errorf("password/reset: failed to mark password reset token as used: %w", err)
	}

	if err := q.UpsertPasswordCredentialByUserID(ctx, query.UpsertPasswordCredentialByUserIDParams{
		ID:                   tok.UserID,
		UserCredentialKey:    user.Email,
		UserCredentialSecret: passwordHash,
	}); err != nil {
		return fmt.Errorf("password/reset: failed to set user password: %w", err)
	}

	// Once password is changed, we need to expire all sessions for this user
	// due to security reasons.
	if _, err := q.ExpireAllSessionsByUserID(ctx, user.ID); err != nil {
		return fmt.Errorf("password/reset: failed to expire sessions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("password/reset: failed to commit transaction: %w", err)
	}

	if err := h.sender.Send(ctx, sender.Message{
		Email:   user.Email,
		Payload: PasswordResetSuccessMessagePayload{},
	}); err != nil {
		return fmt.Errorf("password/reset: failed to send success message: %w", err)
	}

	return nil
}
