package session

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.inout.gg/common/authentication/db/driver"
	"go.inout.gg/common/authentication/internal/query"
	"go.inout.gg/common/authentication/strategy"
	"go.inout.gg/common/authentication/user"
	"go.inout.gg/common/http/cookie"
	"go.inout.gg/common/must"
	"go.inout.gg/common/random"
	"go.inout.gg/common/sql/dbutil"
	"go.inout.gg/common/uuidv7"
)

var _ strategy.Authenticator[any] = (*session[any])(nil)

var (
	DefaultCookieName = "usid"
)

type session[T any] struct {
	driver driver.Driver
	config *Config
}

type Config struct {
	CookieName string
	ExpiresIn  time.Duration
}

// New creates a new session authenticator.
//
// The sesion authenticator uses a DB to store sessions and a cookie to
// store the session ID.
func New[T any](driver driver.Driver, config *Config) strategy.Authenticator[T] {
	return &session[T]{driver, config}
}

func (s *session[T]) Issue(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := s.driver.Queries()

	sessionID := uuidv7.Must()
	token := must.Must(random.SecureHexString(64))
	if _, err := q.CreateUserSession(ctx, query.CreateUserSessionParams{
		ID:        sessionID,
		Token:     token,
		ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(s.config.ExpiresIn), Valid: true},
	}); err != nil {
		return fmt.Errorf("authentication/session: failed to create session: %w", err)
	}

	cookie.Set(
		w,
		s.config.CookieName,
		must.Must(s.encode(token)),
		cookie.WithHttpOnly,
		cookie.WithExpiresIn(s.config.ExpiresIn),
	)

	return nil
}

func (s *session[T]) Authenticate(
	w http.ResponseWriter,
	r *http.Request,
) (*strategy.User[T], error) {
	ctx := r.Context()
	val := cookie.Get(r, DefaultCookieName)
	if val == "" {
		return nil, user.ErrUnauthorizedUser
	}

	val, err := s.decode(val)
	if err != nil {
		return nil, fmt.Errorf("authentication/session: failed to decode session: %w", err)
	}

	tx, err := s.driver.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication/session: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := tx.Queries()
	_, err = q.FindUserSessionByID(ctx, uuid.UUID{})
	if err != nil {
		if dbutil.IsNotFoundError(err) {
			return nil, user.ErrUnauthorizedUser
		}

		return nil, fmt.Errorf("authentication/session: failed to find user session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("authentication/session: failed to commit transaction: %w", err)
	}

	return nil, nil
}

func (s *session[T]) encode(val string) (string, error) {
	bytes := []byte(val)
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *session[T]) decode(val string) (string, error) {
	bytes, err := base64.URLEncoding.DecodeString(val)
	if err != nil {
		return "", fmt.Errorf("authentication/session: failed to decode cookie: %w", err)
	}

	return string(bytes), nil
}
