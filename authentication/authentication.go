package authentication

import (
	"net/http"

	"github.com/atcirclesquare/common/authentication/password"
	"github.com/atcirclesquare/common/authentication/routes"
	"github.com/atcirclesquare/common/authentication/sso"
	"github.com/go-chi/chi/v5"
)

type Config struct {
	SSOProviders     []sso.Provider[any]
	EmailAndPassword password.EmailAndPasswordProvider
	*routes.Config
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

	for _, provider := range cfg.SSOProviders {
		r := provider.Routes(cfg.Config)
		if r != nil {
			router = r.Apply(router)
		}
	}

	return router
}
