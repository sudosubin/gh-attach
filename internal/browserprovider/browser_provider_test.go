package browserprovider

import (
	"context"
	"net/http"
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestNewDefaultRegistry(t *testing.T) {
	t.Parallel()

	reg := NewDefaultRegistry()

	if got := reg[cookies.BrowserChrome].BackendName(); got != "sweetcookie" {
		t.Fatalf("chrome backend = %q", got)
	}
	if got := reg[cookies.BrowserChromium].BackendName(); got != "sweetcookie" {
		t.Fatalf("chromium backend = %q", got)
	}
	if got := reg[cookies.BrowserFirefox].BackendName(); got != "kooky" {
		t.Fatalf("firefox backend = %q", got)
	}
	if got := reg[cookies.BrowserSafari].BackendName(); got != "kooky" {
		t.Fatalf("safari backend = %q", got)
	}
}

func TestUserAgentForBrowser(t *testing.T) {
	t.Parallel()

	ff1, err := UserAgentForBrowser(cookies.BrowserFirefox)
	if err != nil || ff1 == "" {
		t.Fatalf("firefox user agent error = %v, user-agent = %q", err, ff1)
	}
	ff2, err := UserAgentForBrowser(cookies.BrowserFirefox)
	if err != nil {
		t.Fatalf("firefox user agent error = %v", err)
	}
	if ff1 != ff2 {
		t.Fatalf("firefox user agent should be deterministic; got %q and %q", ff1, ff2)
	}

	if got, err := UserAgentForBrowser(cookies.BrowserSafari); err != nil || got == "" {
		t.Fatalf("safari user agent error = %v, user-agent = %q", err, got)
	}
	autoUA, err := UserAgentForBrowser(cookies.BrowserAuto)
	if err != nil || autoUA == "" {
		t.Fatalf("auto user agent error = %v, user-agent = %q", err, autoUA)
	}
}

func TestBackendName_Unconfigured(t *testing.T) {
	t.Parallel()

	p := &browserProvider{browser: cookies.BrowserChrome}
	if got := p.BackendName(); got != "unconfigured" {
		t.Fatalf("BackendName() = %q, want %q", got, "unconfigured")
	}
}

type stubBackend struct {
	name string
	load func(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error)
}

func (b stubBackend) Name() string { return b.name }

func (b stubBackend) Load(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error) {
	return b.load(ctx, host, source)
}

func TestBrowserProviderLoad_ReturnsCookiesAndUserAgent(t *testing.T) {
	t.Parallel()

	called := false
	p := &browserProvider{
		browser: cookies.BrowserFirefox,
		backend: stubBackend{
			name: "stub",
			load: func(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error) {
				called = true
				if source.Browser != cookies.BrowserFirefox {
					t.Fatalf("source.Browser = %q, want %q", source.Browser, cookies.BrowserFirefox)
				}
				return []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}}, nil
			},
		},
	}

	session, err := p.Load(context.Background(), "github.com", cookies.Source{Browser: cookies.BrowserAuto})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !called {
		t.Fatalf("backend.Load was not called")
	}
	if session.Browser != cookies.BrowserFirefox {
		t.Fatalf("session.Browser = %q, want %q", session.Browser, cookies.BrowserFirefox)
	}
	if len(session.Cookies) != 1 {
		t.Fatalf("len(session.Cookies) = %d, want 1", len(session.Cookies))
	}
	wantUA, _ := UserAgentForBrowser(cookies.BrowserFirefox)
	if session.UserAgent != wantUA {
		t.Fatalf("session.UserAgent = %q", session.UserAgent)
	}
}

func TestBrowserProviderLoad_FailsOnAutoProviderBrowser(t *testing.T) {
	t.Parallel()

	p := &browserProvider{
		browser: cookies.BrowserAuto,
		backend: stubBackend{
			name: "stub",
			load: func(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error) {
				return []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}}, nil
			},
		},
	}

	_, err := p.Load(context.Background(), "github.com", cookies.Source{})
	if err == nil {
		t.Fatalf("Load() error = nil, want non-nil")
	}
}
