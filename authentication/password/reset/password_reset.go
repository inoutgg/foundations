package passwordreset

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.inout.gg/common/authentication/internal/query"
	"go.inout.gg/common/authentication/password"
	"go.inout.gg/common/authentication/sender"
	"go.inout.gg/common/authentication/user"
	"go.inout.gg/common/must"
	"go.inout.gg/common/pointer"
	"go.inout.gg/common/random"
	"go.inout.gg/common/uuidv7"
)

var (
	ErrAuthorizedUser         = fmt.Errorf("password/reset: authorized user access")
	ErrUsedPasswordResetToken = fmt.Errorf("password/reset: used password reset token")
)

const (
	TokenExpiry = 12 * time.Hour
	TokenLength = 32
)

type Config struct {
	TokenLength   int
	TokenExpiryIn time.Duration
}

// ResetTokenMessagePayload is the payload for the reset token message.
type PasswordResetRequestMessagePayload struct {
	Token string
}

// PasswordResetSuccessMessagePayload is the payload for the password reset success message.
type PasswordResetSuccessMessagePayload struct{}

type Handler struct {
	config *Config
	logger slog.Logger
	pool   pgxpool.Pool

	password.PasswordHasher
	query.Queries
	sender.Sender[any]
}

// HandlePasswordResetRequest handles a password reset request.
func (h *Handler) HandlePasswordResetRequest(
	ctx context.Context,
	email string,
) error {
	// Forbid authorized user access.
	usr := user.FromContext[any](ctx)
	if usr != nil {
		return ErrAuthorizedUser
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("password/reset: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := h.Queries.WithTx(tx)
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

	if err := h.Sender.Send(ctx, sender.Message[any]{
		Email: user.Email,
		Name:  "",
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
	// Hash password upfront to avoid unnecessary database TX delay.
	passwordHash, err := h.PasswordHasher.Hash(password)
	if err != nil {
		return fmt.Errorf("password/reset: failed to hash password: %w", err)
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("password/reset: failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	q := h.Queries.WithTx(tx)

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

	if err = q.SetUserPasswordByID(ctx, query.SetUserPasswordByIDParams{
		ID:           tok.UserID,
		PasswordHash: pointer.FromValue(passwordHash),
	}); err != nil {
		return fmt.Errorf("password/reset: failed to set user password: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("password/reset: failed to commit transaction: %w", err)
	}

	if err := h.Sender.Send(ctx, sender.Message[any]{
		Email: user.Email,
		// TODO: add name to user.
		Name:    "",
		Payload: PasswordResetSuccessMessagePayload{},
	}); err != nil {
		return fmt.Errorf("password/reset: failed to send success message: %w", err)
	}

	return nil
}
