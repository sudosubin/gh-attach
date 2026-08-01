package app

import (
	"context"
	"fmt"
	"sync"

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

// Resolve starts login resolution in the background: it and cookie loading
// are independent until a candidate browser session's dotcom_user actually
// needs to be compared against the resolved login, so there's no reason for
// cookie loading to wait for login to finish first. The two only rendezvous
// (via the ghLogin thunk passed to CookieResolver.Resolve) at that point,
// and not at all if no session ever has a dotcom_user to check.
func (s *SessionResolver) Resolve(ctx context.Context, host string, sources []cookies.Source) (ResolvedCookies, error) {
	type loginResult struct {
		login string
		err   error
	}

	loginCh := make(chan loginResult, 1)
	go func() {
		login, err := s.logins.Login(host)
		loginCh <- loginResult{login, err}
	}()

	ghLogin := sync.OnceValues(func() (string, error) {
		r := <-loginCh
		if r.err != nil {
			return "", fmt.Errorf("resolve current login: %w", r.err)
		}
		return r.login, nil
	})

	return s.cookies.Resolve(ctx, host, ghLogin, sources)
}
