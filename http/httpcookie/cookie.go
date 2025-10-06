package httpcookie

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
	Domain    string
	Path      string
	ExpiresIn time.Duration
	SameSite  http.SameSite
	Secure    bool
	HTTPOnly  bool
}

// WithExpiresIn sets the ExpiresIn option on the cookie.
func WithExpiresIn(expiresIn time.Duration) func(*Option) {
	return func(opt *Option) { opt.ExpiresIn = expiresIn }
}

// WithHTTPOnly sets the HttpOnly flag on the cookie.
func WithHTTPOnly(opt *Option) { opt.HTTPOnly = true }

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
	//nolint:exhaustruct
	opt := Option{
		SameSite: http.SameSiteDefaultMode,
		Path:     "/",
	}

	for _, o := range options {
		o(&opt)
	}

	//nolint:exhaustruct
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     opt.Path,
		Domain:   opt.Domain,
		HttpOnly: opt.HTTPOnly,
		SameSite: opt.SameSite,
	}
	if opt.ExpiresIn != 0 {
		cookie.Expires = time.Now().Add(opt.ExpiresIn)
	}

	http.SetCookie(w, cookie)
}

// Delete deletes the cookie with the given name if it exists.
func Delete(w http.ResponseWriter, r *http.Request, name string) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return
	}

	http.SetCookie(
		w,
		//nolint:exhaustruct
		&http.Cookie{
			Name:     name,
			Value:    "",
			MaxAge:   -1,
			HttpOnly: cookie.HttpOnly,
			SameSite: cookie.SameSite,
		},
	)
}
