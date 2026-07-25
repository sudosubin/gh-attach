package app

import (
	"context"
	"fmt"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

// LoginResolver resolves the login the upload acts as, used to pick the matching cookie container.
type LoginResolver interface {
	Login(host string) (string, error)
}

// apiLoginResolver is the REST fallback, satisfied by rest.Service.CurrentLogin.
type apiLoginResolver interface {
	CurrentLogin() (string, error)
}

type configLoginResolver struct {
	api apiLoginResolver
}

func NewConfigLoginResolver(api apiLoginResolver) LoginResolver {
	return configLoginResolver{api: api}
}

func (r configLoginResolver) Login(host string) (string, error) {
	if login, ok := loginFromLocalToken(host); ok {
		return login, nil
	}
	return r.api.CurrentLogin()
}

// SessionResolver hides the login-then-cookie-match coupling behind a single "give me a session" call.
type SessionResolver struct {
	logins  LoginResolver
	cookies *CookieResolver
}

func NewSessionResolver(logins LoginResolver, cookies *CookieResolver) *SessionResolver {
	return &SessionResolver{logins: logins, cookies: cookies}
}

func (s *SessionResolver) Resolve(ctx context.Context, host string, sources []cookies.Source) (ResolvedCookies, error) {
	login, err := s.logins.Login(host)
	if err != nil {
		return ResolvedCookies{}, fmt.Errorf("resolve current login: %w", err)
	}
	return s.cookies.Resolve(ctx, host, login, sources)
}
