package cookie

import (
	"net/http"
	"time"
)

// Get returns the value of the cookie with the given name. If the
// cookie is not found, the empty string is returned.
func Get(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}

	return cookie.Value
}

// Option is a set of options for setting a cookie.
type Option struct {
	ExpiresIn time.Duration
	Domain    string
	Secure    bool
	HttpOnly  bool
	SameSite  http.SameSite
	Path      string
}

// WithExpiresIn sets the ExpiresIn option on the cookie.
func WithExpiresIn(expiresIn time.Duration) func(*Option) {
	return func(opt *Option) { opt.ExpiresIn = expiresIn }
}

// WithHttpOnly sets the HttpOnly flag on the cookie.
func WithHttpOnly(opt *Option) { opt.HttpOnly = true }

// WithSecure sets the Secure flag on the cookie.
func WithSecure(opt *Option) { opt.Secure = true }

// WithSameSite sets the SameSite flag on the cookie.
func WithSameSite(sameSite http.SameSite) func(*Option) {
	return func(opt *Option) { opt.SameSite = sameSite }
}

// WithDomain sets the Domain flag on the cookie.
func WithDomain(domain string) func(*Option) {
	return func(opt *Option) { opt.Domain = domain }
}

// Set sets the cookie with the given name and value.
func Set(w http.ResponseWriter, name, value string, options ...func(*Option)) {
	opt := Option{
		SameSite: http.SameSiteDefaultMode,
	}

	for _, o := range options {
		o(&opt)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     opt.Path,
		Domain:   opt.Domain,
		HttpOnly: opt.HttpOnly,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(opt.ExpiresIn),
	})
}

// Delete deletes the cookie with the given name if it exists.
func Delete(w http.ResponseWriter, r *http.Request, name string) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     name,
			Value:    "",
			MaxAge:   -1,
			HttpOnly: cookie.HttpOnly,
			SameSite: cookie.SameSite,
		},
	)
}
