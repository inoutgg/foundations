package csrf

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.inout.gg/common/random"
)

var (
	DefaultFieldName  = "csrf_token"
	DefaultHeaderName = "X-CSRF-Token"
	DefaultCookieName = "csrf_token"
)

var (
	ErrInvalidToken  = errors.New("authentication/csrf: invalid token")
	ErrTokenMismatch = errors.New("authentication/csrf: token mismatch")
	ErrTokenNotFound = errors.New("authentication/csrf: token not found")
)

type TokenOption struct {
	ChecksumSecret string
	TokenLength    int

	HeaderName     string // optional (default: "X-CSRF-Token")
	FieldName      string // optional (default: "csrf_token")
	CookieName     string // optional (default: "csrf_token")
	CookieSecure   bool
	CookieSameSite http.SameSite
}

// WithChecksumSecret sets the checksum secret on the token option.
func WithChecksumSecret(secret string) func(*TokenOption) {
	return func(opt *TokenOption) {
		opt.ChecksumSecret = secret
	}
}

func NewTokenOption(opts ...func(*TokenOption)) *TokenOption {
	opt := TokenOption{
		HeaderName:     DefaultHeaderName,
		FieldName:      DefaultFieldName,
		CookieName:     DefaultCookieName,
		TokenLength:    32,
		CookieSameSite: http.SameSiteLaxMode,
	}
	for _, f := range opts {
		f(&opt)
	}

	return &opt
}

func (opt *TokenOption) cookieName() string {
	name := opt.CookieName
	if opt.CookieSecure {
		name = fmt.Sprintf("__Secure-%s", name)
	}

	return name
}

// Token implements CSRF token using the double submit cookie pattern.
type Token struct {
	value    string
	checksum string
	option   *TokenOption
}

// newToken returns a new CSRF token.
// An error is returned if the token cannot be generated.
func newToken(opt *TokenOption) (*Token, error) {
	val, err := random.SecureHexString(opt.TokenLength)
	if err != nil {
		return nil, err
	}

	checksum := computeChecksum(val, opt.ChecksumSecret)
	return &Token{
		value:    val,
		checksum: checksum,
		option:   opt,
	}, nil
}

// fromRequest returns a CSRF token from an HTTP request by reading a cookie.
func fromRequest(r *http.Request, opt *TokenOption) (*Token, error) {
	cookie, err := r.Cookie(opt.cookieName())
	if err != nil {
		return nil, fmt.Errorf("authentication/csrf: unable to retrieve cookie: %w", err)
	}

	parts := strings.Split(cookie.Value, "|")
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}

	tok := &Token{
		value:    parts[0],
		checksum: parts[1],
		option:   opt,
	}
	if !tok.validateChecksum() {
		return nil, ErrInvalidToken
	}

	return tok, nil
}

// validateRequest returns true if the HTTP request contains a valid CSRF token.
func validateRequest(r *http.Request, opt *TokenOption) error {
	tok, err := fromRequest(r, opt)
	if err != nil {
		return err
	}

	return tok.validateRequest(r)
}

func (t *Token) validateChecksum() bool {
	expectedChecksum := computeChecksum(t.value, t.option.ChecksumSecret)
	return t.checksum == expectedChecksum
}

func (t *Token) validateRequest(r *http.Request) error {
	opt := t.option

	// Get the CSRF token from the header or the form field.
	tokValue := r.Header.Get(opt.HeaderName)
	if tokValue == "" {
		tokValue = r.PostFormValue(opt.FieldName)
	}

	// If the token is missing, try to get it from the multipart form.
	if tokValue == "" && r.MultipartForm != nil {
		vals := r.MultipartForm.Value[opt.FieldName]
		if len(vals) > 0 {
			tokValue = vals[0]
		}
	}

	if t.value != tokValue {
		return ErrTokenMismatch
	}

	return nil
}

// String returns the CSRF token value.
func (t *Token) String() string {
	return t.value
}

// Cookie returns an HTTP cookie containing the CSRF token.
func (t *Token) Cookie() *http.Cookie {
	val := fmt.Sprintf("%s|%s", t.value, t.checksum)
	cookie := http.Cookie{
		Name:     t.option.cookieName(),
		Value:    val,
		HttpOnly: true,
		Secure:   t.option.CookieSecure,
		SameSite: t.option.CookieSameSite,
	}

	return &cookie
}

// computeChecksum return the sha256 checksum of the given value and secret.
func computeChecksum(val, secret string) string {
	cs := sha256.Sum256([]byte(fmt.Sprintf("%s%s", val, secret)))
	return fmt.Sprintf("%x", cs)
}
