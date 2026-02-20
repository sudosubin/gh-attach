package browserprovider

import (
	"context"
	"net/http"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

// Backend is an infrastructure adapter backed by a concrete cookie library
// such as sweetcookie or kooky.
type Backend interface {
	Name() string
	Load(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error)
}

type BrowserSession struct {
	Browser   cookies.Browser
	Cookies   []*http.Cookie
	UserAgent string
}

// BrowserProvider is a browser-specific cookie provider.
type BrowserProvider interface {
	Browser() cookies.Browser
	BackendName() string
	Load(ctx context.Context, host string, source cookies.Source) (BrowserSession, error)
}
