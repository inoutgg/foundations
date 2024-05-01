package password

import (
	"net/http"

	"github.com/atcirclesquare/common/authentication/routes"
	"github.com/atcirclesquare/common/http/errorhandler"
	"github.com/atcirclesquare/common/http/routerutil"
	"github.com/go-chi/chi/v5"
)

var _ routerutil.Applicator = (*routesAplicator)(nil)

type routesAplicator struct {
	*routes.Config
	*EmailAndPasswordProvider
}

func (r *routesAplicator) Apply(router chi.Router) chi.Router {
	withError := errorhandler.WithErrorHandler(r.Config.ErrorHandler)

	router.Post(
		"/password/reset",
		withError(errorhandler.HandlerFunc(r.resetPassword)),
	)

	return router
}

func (r *routesAplicator) resetPassword(w http.ResponseWriter, req *http.Request) error {
	return nil
}
