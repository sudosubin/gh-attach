package browserprovider

import (
	"context"
	"net/http"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

type CookieSet struct {
	Profile string
	Cookies []*http.Cookie
}

type Backend interface {
	Name() string
	Load(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error)
}

type BrowserSession struct {
	Browser   cookies.Browser
	Profile   string
	Cookies   []*http.Cookie
	UserAgent string
}

type BrowserProvider interface {
	Browser() cookies.Browser
	BackendName() string
	Load(ctx context.Context, host string, source cookies.Source) ([]BrowserSession, error)
}
