package csrf

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.inout.gg/common/random"
)

var (
	ErrInvalidToken  = errors.New("authentication/csrf: invalid token")
	ErrTokenMismatch = errors.New("authentication/csrf: token mismatch")
	ErrTokenNotFound = errors.New("authentication/csrf: token not found")
)

type tokenConfig struct {
	ChecksumSecret string
	TokenLength    int

	HeaderName     string // optional (default: "X-CSRF-Token")
	FieldName      string // optional (default: "csrf_token")
	CookieName     string // optional (default: "csrf_token")
	CookieSecure   bool
	CookieSameSite http.SameSite
}

func (opt *tokenConfig) cookieName() string {
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
	config   *tokenConfig
}

// newToken returns a new CSRF token.
// An error is returned if the token cannot be generated.
func newToken(opt *tokenConfig) (*Token, error) {
	val, err := random.SecureHexString(opt.TokenLength)
	if err != nil {
		return nil, err
	}

	checksum := computeChecksum(val, opt.ChecksumSecret)

	return &Token{
		value:    val,
		checksum: checksum,
		config:   opt,
	}, nil
}

// fromRequest returns a CSRF token from an HTTP request by reading a cookie.
func fromRequest(r *http.Request, opt *tokenConfig) (*Token, error) {
	cookie, err := r.Cookie(opt.cookieName())
	if err != nil {
		return nil, fmt.Errorf("authentication/csrf: unable to retrieve cookie: %w", err)
	}

	value, checksum, err := decodeCookieValue(cookie.Value)
	if err != nil {
		return nil, err
	}

	tok := &Token{
		value:    value,
		checksum: checksum,
		config:   opt,
	}

	if !tok.validateChecksum() {
		return nil, ErrInvalidToken
	}

	return tok, nil
}

// validateRequest returns true if the HTTP request contains a valid CSRF token.
func validateRequest(r *http.Request, opt *tokenConfig) error {
	tok, err := fromRequest(r, opt)
	if err != nil {
		return err
	}

	return tok.validateRequest(r)
}

func (t *Token) validateChecksum() bool {
	expectedChecksum := computeChecksum(t.value, t.config.ChecksumSecret)
	return t.checksum == expectedChecksum
}

func (t *Token) validateRequest(r *http.Request) error {
	opt := t.config

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
	cookie := http.Cookie{
		Name:     t.config.cookieName(),
		Value:    t.cookieValue(),
		HttpOnly: true,
		Secure:   t.config.CookieSecure,
		SameSite: t.config.CookieSameSite,
	}

	return &cookie
}

func (t *Token) cookieValue() string {
	content := []byte(fmt.Sprintf("%s|%s", t.value, t.checksum))
	return base64.URLEncoding.EncodeToString(content)
}

func decodeCookieValue(val string) (string, string, error) {
	bytes, err := base64.URLEncoding.DecodeString(val)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	content := string(bytes)
	parts := strings.Split(content, "|")
	if len(parts) != 2 {
		return "", "", ErrInvalidToken
	}

	return parts[0], parts[1], nil
}

// computeChecksum return the sha256 checksum of the given value and secret.
func computeChecksum(val, secret string) string {
	cs := sha256.Sum256([]byte(fmt.Sprintf("%s%s", val, secret)))
	return hex.EncodeToString(cs[:])
}
