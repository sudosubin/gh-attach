package app

import (
	"errors"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/web"
)

func newTokenSession(host, value string) (web.Session, error) {
	token := strings.TrimSpace(value)
	if token == "" {
		return web.Session{}, errors.New("session token is empty")
	}
	lowerToken := strings.ToLower(token)
	if strings.HasPrefix(lowerToken, "cookie:") || strings.Contains(token, ";") {
		return web.Session{}, errors.New("session token must be a bare user_session value")
	}
	if name, _, ok := strings.Cut(token, "="); ok &&
		(strings.EqualFold(name, "user_session") || strings.EqualFold(name, "__Host-user_session_same_site")) {
		return web.Session{}, errors.New("session token must not include a cookie name")
	}

	domain := (&url.URL{Host: host}).Hostname()
	cookie := &http.Cookie{
		Name:     "user_session",
		Value:    token,
		Path:     "/",
		Domain:   domain,
		Secure:   true,
		HttpOnly: true,
	}
	if domain == "" || cookie.Valid() != nil {
		return web.Session{}, errors.New("session token is invalid")
	}
	sameSiteCookie := *cookie
	sameSiteCookie.Name = "__Host-user_session_same_site"

	return web.Session{
		Cookies: []*http.Cookie{cookie, &sameSiteCookie},
		UserAgent: browserprovider.UserAgent(
			cookies.BrowserChrome,
			runtime.GOOS,
			"",
		),
	}, nil
}
