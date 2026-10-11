package app

import (
	"context"
	"fmt"
	"net/url"
	"sync"

	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/web"
)

// SessionResolver hides the login-then-cookie-match coupling behind a single "give me a session" call.
type SessionResolver struct {
	logins  LoginResolver
	cookies *CookieResolver
}

func NewSessionResolver(logins LoginResolver, cookies *CookieResolver) *SessionResolver {
	return &SessionResolver{logins: logins, cookies: cookies}
}

// Resolve runs login in the background so it overlaps cookie loading.
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

func (s *Service) resolveBrowserSession(
	ctx context.Context,
	host string,
	input cookieSourceInput,
	verbose bool,
) (web.Session, error) {
	sources, err := resolveCookieSources(input)
	if err != nil {
		return web.Session{}, err
	}
	loginResolver := s.loginResolver
	if loginResolver == nil {
		loginResolver = NewConfigLoginResolver(lazyAPILoginResolver{host: host})
	}
	resolved, err := NewSessionResolver(
		loginResolver,
		NewCookieResolver(s.providers, verbose, s.stderr),
	).Resolve(ctx, (&url.URL{Host: host}).Hostname(), sources)
	if err != nil {
		return web.Session{}, err
	}
	return web.Session{
		Cookies:   resolved.Session.Cookies,
		UserAgent: resolved.Session.UserAgent,
	}, nil
}
