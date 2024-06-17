package session

import (
	"net/http"

	"go.inout.gg/common/authentication"
	"go.inout.gg/common/authentication/db/driver"
	"go.inout.gg/common/authentication/user"
	"go.inout.gg/common/http/cookie"
	httperror "go.inout.gg/common/http/error"
	"go.inout.gg/common/internal/uuidv7"
)

// LogoutHandler is a handler that logs out the user and deletes the session.
type LogoutHandler struct {
	driver driver.Driver
	config *Config
}

func NewLogoutHandler(driver driver.Driver, config *Config) *LogoutHandler {
	return &LogoutHandler{driver, config}
}

func (h *LogoutHandler) HandleLogout(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	q := h.driver.Queries()

	usr := user.FromRequest[any](r)
	if usr == nil {
		return httperror.FromError(authentication.ErrUnauthorizedUser, http.StatusUnauthorized)
	}

	if _, err := q.ExpireSessionByID(ctx, uuidv7.ToPgxUUID(usr.ID)); err != nil {
		return httperror.FromError(err, http.StatusInternalServerError)
	}

	// Delete session cookie.
	cookie.Delete(w, r, h.config.CookieName)

	return nil
}
