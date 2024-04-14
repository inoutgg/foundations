package cookie

import (
	"net/http"
	"time"
)

var _ Cookie = (*cookie)(nil)

type CookieOption struct {
	SameSite http.SameSite
	// Controls whether cookie can be accessed through JavaScript.
	HttpOnly  bool
	ExpiresIn time.Duration
	Path      string
}

type Cookie interface {
	// Set sets a cookie with the given name and value.
	Set(string, string, ...func(*CookieOption))

	// Get returns the value of the cookie with the given name. If the
	Get(string) string

	// Delete deletes the cookie with the given name.
	Delete(string, ...func(*CookieOption))
}

type cookie struct {
	req    *http.Request
	resp   http.ResponseWriter
	secure bool
}

func (c *cookie) Set(name string, value string, cfg ...func(*CookieOption)) {
	config := newConfig(cfg...)

	var expiresAt time.Time
	if config.ExpiresIn != 0 {
		expiresAt = time.Now().Add(config.ExpiresIn * time.Second)
	}

	http.SetCookie(c.resp, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: config.HttpOnly,
		Secure:   c.secure,
		SameSite: config.SameSite,
		Expires:  expiresAt,
		Path:     config.Path,
	})
}

func (c *cookie) Get(name string) string {
	cookie, err := c.req.Cookie(name)
	if err != nil {
		return ""
	}

	return cookie.Value
}

func (c *cookie) Delete(name string, cfg ...func(*CookieOption)) {
	config := newConfig(cfg...)
	http.SetCookie(c.resp, &http.Cookie{
		Name:     name,
		Value:    "",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: config.SameSite,
		MaxAge:   -1,
	})
}

// newConfig creates a new CookieOption from the given functions.
func newConfig(cfg ...func(*CookieOption)) CookieOption {
	config := CookieOption{
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	}
	for _, f := range cfg {
		f(&config)
	}

	return config
}
