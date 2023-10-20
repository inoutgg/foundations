package csrf

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type Option struct {
	ChecksumSecret string
	TokenLength    int

	HeaderName string
	FieldName  string

	CookieName     string
	CookieSecure   bool
	CookieSameSite http.SameSite
}

func NewOption(f ...func(*Option)) {
	opt := &Option{}
	for _, v := range f {
		v(opt)
	}
}

// Token implements CSRF using the double submit cookie pattern.
type Token struct {
	value    string
	checksum string
	option   *Option
}

// New returns a new CSRF token.
// An error is returned if the token cannot be generated.
func New(opt *Option) (*Token, error) {
	val, err := randomHexString(opt.TokenLength)
	if err != nil {
		return nil, err
	}
	checksum := sha256.Sum256([]byte(fmt.Sprintf("%x%s", val, opt.ChecksumSecret)))

	return &Token{
		value:    val,
		checksum: string(checksum[:]),
		option:   opt,
	}, nil
}

// FromCookie returns a CSRF token from an HTTP request cookie.
func FromCookie(r *http.Request, opt *Option) (*Token, error) {
	cookie, err := r.Cookie(opt.CookieName)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(cookie.Value, "|")
	if len(parts) != 2 {
		return nil, errors.New("invalid token")
	}

	tok := &Token{
		value:    parts[0],
		checksum: parts[1],
		option:   opt,
	}
	if !tok.validateChecksum() {
		return nil, errors.New("invalid token")
	}

	return tok, nil
}

// ValidateRequest returns true if the HTTP request contains a valid CSRF token.
func ValidateRequest(r *http.Request, opt *Option) bool {
	tok, err := FromCookie(r, opt)
	if err != nil {
		return false
	}

	return !tok.validateRequest(r)
}

func (t *Token) validateChecksum() bool {
	checksum := sha256.Sum256([]byte(fmt.Sprintf("%x%s", t.value, t.option.ChecksumSecret)))
	return t.checksum == string(checksum[:])
}

func (t *Token) validateRequest(r *http.Request) bool {
	opt := t.option
	tokValue := r.Header.Get(opt.HeaderName)
	if tokValue == "" {
		tokValue = r.PostFormValue(opt.FieldName)
	}

	if tokValue == "" && r.MultipartForm != nil {
		vals := r.MultipartForm.Value[opt.FieldName]
		if len(vals) > 0 {
			tokValue = vals[0]
		}
	}

	return t.value == tokValue
}

// Value returns the CSRF token value.
func (t *Token) Value() string {
	return t.value
}

// Cookie returns an HTTP cookie containing the CSRF token.
func (t *Token) Cookie() *http.Cookie {
	val := fmt.Sprintf("%s|%s", t.value, t.checksum)
	name := t.option.CookieName
	if t.option.CookieSecure {
		name = fmt.Sprintf("__Secure-%s", name)
	}

	cookie := http.Cookie{
		Name:     t.option.CookieName,
		Value:    val,
		HttpOnly: true,
		Secure:   t.option.CookieSecure,
		SameSite: t.option.CookieSameSite,
	}

	return &cookie
}

func parseTokenValue(r *http.Request, opt *Option) string {
	tokValue := r.Header.Get(opt.HeaderName)
	if tokValue == "" {
		tokValue = r.PostFormValue(opt.FieldName)
	}

	if tokValue == "" && r.MultipartForm != nil {
		vals := r.MultipartForm.Value[opt.FieldName]
		if len(vals) > 0 {
			tokValue = vals[0]
		}
	}

	return tokValue
}

func randomHexString(l int) (string, error) {
	bytes := make([]byte, l)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", bytes), nil
}
