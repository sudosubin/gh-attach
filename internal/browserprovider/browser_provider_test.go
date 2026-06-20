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

	for _, b := range cookies.ConcreteBrowsers() {
		p, ok := reg[b]
		if !ok {
			t.Fatalf("browser %q missing from registry", b)
		}
		if got := p.BackendName(); got != "sweetcookie" {
			t.Fatalf("%q backend = %q, want sweetcookie", b, got)
		}
	}

	if _, ok := reg[cookies.BrowserAuto]; ok {
		t.Fatalf("auto should not be registered")
	}
	if len(reg) != len(cookies.ConcreteBrowsers()) {
		t.Fatalf("registry has %d entries, want %d", len(reg), len(cookies.ConcreteBrowsers()))
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

	for _, b := range cookies.ConcreteBrowsers() {
		got, err := UserAgentForBrowser(b)
		if err != nil || got == "" {
			t.Fatalf("browser=%s user agent error = %v, user-agent = %q", b, err, got)
		}
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
	load func(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error)
}

func (b stubBackend) Name() string { return b.name }

func (b stubBackend) Load(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
	return b.load(ctx, host, source)
}

func TestBrowserProviderLoad_ReturnsSessionPerSet(t *testing.T) {
	t.Parallel()

	called := false
	p := &browserProvider{
		browser: cookies.BrowserFirefox,
		backend: stubBackend{
			name: "stub",
			load: func(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
				called = true
				if source.Browser != cookies.BrowserFirefox {
					t.Fatalf("source.Browser = %q, want %q", source.Browser, cookies.BrowserFirefox)
				}
				return []CookieSet{
					{Profile: "default:sudosubin@gmail.com", Cookies: []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}}},
					{Profile: "default:sudosubin@example.com", Cookies: []*http.Cookie{{Name: "dotcom_user", Value: "octocat"}}},
				}, nil
			},
		},
	}

	sessions, err := p.Load(t.Context(), "github.com", cookies.Source{Browser: cookies.BrowserAuto})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !called {
		t.Fatalf("backend.Load was not called")
	}
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(sessions))
	}
	wantUA, _ := UserAgentForBrowser(cookies.BrowserFirefox)
	for i, s := range sessions {
		if s.Browser != cookies.BrowserFirefox {
			t.Fatalf("session[%d].Browser = %q", i, s.Browser)
		}
		if s.UserAgent != wantUA {
			t.Fatalf("session[%d].UserAgent = %q", i, s.UserAgent)
		}
	}
	if sessions[0].Profile != "default:sudosubin@gmail.com" || sessions[1].Profile != "default:sudosubin@example.com" {
		t.Fatalf("session profiles = %q,%q", sessions[0].Profile, sessions[1].Profile)
	}
}

func TestBrowserProviderLoad_FailsOnAutoProviderBrowser(t *testing.T) {
	t.Parallel()

	p := &browserProvider{
		browser: cookies.BrowserAuto,
		backend: stubBackend{
			name: "stub",
			load: func(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
				return []CookieSet{{Cookies: []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}}}}, nil
			},
		},
	}

	_, err := p.Load(t.Context(), "github.com", cookies.Source{})
	if err == nil {
		t.Fatalf("Load() error = nil, want non-nil")
	}
}
