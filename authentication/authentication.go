package authentication

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.inout.gg/common/authentication/password"
	"go.inout.gg/common/authentication/sso"
)

type Config struct {
	SSOProviders     []sso.Provider[any]
	EmailAndPassword password.Handler
}

// WithSSOProviders adds the given SSO providers to the authentication handler.
func WithSSOProviders(providers ...sso.Provider[any]) func(*Config) {
	return func(c *Config) {
		c.SSOProviders = providers
	}
}

func Handler(configs ...func(*Config)) http.Handler {
	var router chi.Router = chi.NewRouter()
	var cfg Config
	for _, config := range configs {
		config(&cfg)
	}

	return router
}
