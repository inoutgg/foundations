// SSO implements authentication logic to sign in with OpenID providers.
package sso

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/atcirclesquare/common/authentication/routes"
	"github.com/atcirclesquare/common/must"
	"github.com/atcirclesquare/common/random"
	"golang.org/x/oauth2"
)

type UserInfo[T any] interface {
	Claims() T
	Email() string
}

type Provider[T any] interface {
	routes.Applicator
	ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error)
	UserInfo(ctx context.Context, token *oauth2.Token) (UserInfo[T], error)
	AuthCodeURL(state string) string
}

type ProviderInfo[T any] struct {
	UserInfo UserInfo[T]

	RefreshToken string
	AccessToken  string
	Code         string
}

type ProviderState struct {
	State string
	Nonce string
	URL   string
}

// HandleAuthorize handles the authorization request to the OpenID provider.
func HandleAuthorize[T any](
	ctx context.Context,
	req *http.Request,
	provider Provider[T],
) (*ProviderState, error) {
	state := must.Must(random.SecureRandomHexString(32))
	nonce := must.Must(random.SecureRandomHexString(32))
	url := provider.AuthCodeURL(state)

	return &ProviderState{
		State: state,
		Nonce: nonce,
		URL:   url,
	}, nil
}

// HandleCallback handles the callback from the OpenID provider.
func HandleCallback[T any](
	ctx context.Context,
	req *http.Request,
	provider Provider[T],
) (*ProviderInfo[T], error) {
	q := parseQuery(req)

	extError := q.Get("error")
	if extError != "" {
		return nil, fmt.Errorf("authentication/oauth: external error: %s", extError)
	}

	code := q.Get("code")
	if code == "" {
		return nil, fmt.Errorf("authentication/oauth: missing authentication code")
	}

	token, err := provider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("authentication/oauth: unable to exchange code for token: %w", err)
	}

	userInfo, err := provider.UserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("authentication/oauth: unable to get user info: %w", err)
	}

	return &ProviderInfo[T]{
		UserInfo:     userInfo,
		RefreshToken: token.RefreshToken,
		AccessToken:  token.AccessToken,
		Code:         code,
	}, nil
}

// parseQuery parses the query parameters from the request.
//
// If the request method is GET, the query parameters are parsed from the URL,
// otherwise they are parsed from the request body with a fallback to the URL.
func parseQuery(req *http.Request) url.Values {
	if req.Method == http.MethodGet {
		return req.URL.Query()
	}

	if err := req.ParseForm(); err == nil {
		return req.Form
	}

	return req.URL.Query()
}
