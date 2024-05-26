package session

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"go.inout.gg/common/authentication"
	"go.inout.gg/common/authentication/db/driver"
	"go.inout.gg/common/authentication/internal/query"
	"go.inout.gg/common/authentication/strategy"
	"go.inout.gg/common/http/cookie"
	"go.inout.gg/common/must"
	"go.inout.gg/common/random"
	"go.inout.gg/common/sql/dbutil"
	"go.inout.gg/common/uuidv7"
)

var _ strategy.Authenticator[any] = (*sessionStrategy[any])(nil)

var (
	DefaultCookieName = "usid"
	DefaultExpiresIn  = time.Hour * 12
)

type sessionStrategy[T any] struct {
	driver driver.Driver
	config *Config
}

type Config struct {
	CookieName string        // optinal (default: "usid")
	ExpiresIn  time.Duration // optinal (default: 12h)
}

// New creates a new session authenticator.
//
// The sesion authenticator uses a DB to store sessions and a cookie to
// store the session ID.
func New[T any](driver driver.Driver, config ...func(*Config)) strategy.Authenticator[T] {
	cfg := &Config{
		CookieName: DefaultCookieName,
		ExpiresIn:  DefaultExpiresIn,
	}
	for _, c := range config {
		c(cfg)
	}

	return &sessionStrategy[T]{driver, cfg}
}

func (s *sessionStrategy[T]) Issue(
	w http.ResponseWriter,
	r *http.Request,
	user *strategy.User[T],
) (*strategy.Session[T], error) {
	ctx := r.Context()
	q := s.driver.Queries()

	sessionID := uuidv7.Must()
	expiresAt := time.Now().Add(s.config.ExpiresIn)
	token := must.Must(random.SecureHexString(64))
	if _, err := q.CreateUserSession(ctx, query.CreateUserSessionParams{
		ID:        sessionID,
		UserID:    user.ID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	}); err != nil {
		return nil, fmt.Errorf("authentication/session: failed to create session: %w", err)
	}

	cookie.Set(
		w,
		s.config.CookieName,
		s.encode(token),
		cookie.WithHttpOnly,
		cookie.WithExpiresIn(s.config.ExpiresIn),
	)

	return &strategy.Session[T]{
		ID:        sessionID,
		ExpiresAt: expiresAt,
		T:         nil,
	}, nil
}

func (s *sessionStrategy[T]) Authenticate(
	w http.ResponseWriter,
	r *http.Request,
) (*strategy.Session[T], error) {
	ctx := r.Context()
	sessionIDStr := cookie.Get(r, DefaultCookieName)
	if sessionIDStr == "" {
		return nil, authentication.ErrUnauthorizedUser
	}

	sessionIDStr, err := s.decode(sessionIDStr)
	if err != nil {
		cookie.Delete(w, r, s.config.CookieName)
		return nil, authentication.ErrUnauthorizedUser
	}

	sessionID, err := uuidv7.FromString(sessionIDStr)
	if err != nil {
		cookie.Delete(w, r, s.config.CookieName)
		return nil, authentication.ErrUnauthorizedUser
	}

	tx, err := s.driver.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication/session: failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := tx.Queries()
	session, err := q.FindUserSessionByID(ctx, sessionID)
	if err != nil {
		if dbutil.IsNotFoundError(err) {
			cookie.Delete(w, r, s.config.CookieName)
			return nil, authentication.ErrUnauthorizedUser
		}

		return nil, fmt.Errorf("authentication/session: failed to find user session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("authentication/session: failed to commit transaction: %w", err)
	}

	return &strategy.Session[T]{
		ID:        session.ID,
		ExpiresAt: session.ExpiresAt.Time,
		T:         nil,
	}, nil
}

func (s *sessionStrategy[T]) encode(val string) string {
	bytes := []byte(val)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (s *sessionStrategy[T]) decode(val string) (string, error) {
	bytes, err := base64.URLEncoding.DecodeString(val)
	if err != nil {
		return "", fmt.Errorf("authentication/session: failed to decode cookie: %w", err)
	}

	return string(bytes), nil
}
