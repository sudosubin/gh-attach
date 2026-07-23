package browserprovider

import (
	"context"
	"net/http"
	"runtime"
	"strings"
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
	inline, ok := reg[cookies.BrowserInline]
	if !ok {
		t.Fatalf("inline provider missing from registry")
	}
	if got := inline.BackendName(); got != "sweetcookie" {
		t.Fatalf("inline backend = %q, want sweetcookie", got)
	}
	if len(reg) != len(cookies.ConcreteBrowsers())+1 {
		t.Fatalf("registry has %d entries, want %d", len(reg), len(cookies.ConcreteBrowsers())+1)
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
			load: func(_ context.Context, _ string, source cookies.Source) ([]CookieSet, error) {
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
	wantUA := UserAgent(cookies.BrowserFirefox, runtime.GOOS, "")
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
			load: func(_ context.Context, _ string, _ cookies.Source) ([]CookieSet, error) {
				return []CookieSet{{Cookies: []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}}}}, nil
			},
		},
	}

	_, err := p.Load(t.Context(), "github.com", cookies.Source{})
	if err == nil {
		t.Fatalf("Load() error = nil, want non-nil")
	}
}

func TestBrowserProviderLoad_InlineUsesFallbackUserAgent(t *testing.T) {
	t.Parallel()

	path := writeInlineCookieFile(
		t,
		`[{"name":"dotcom_user","value":"octocat","domain":".github.com","path":"/"}]`,
	)
	provider := NewDefaultRegistry()[cookies.BrowserInline]

	sessions, err := provider.Load(
		t.Context(),
		"github.com",
		cookies.Source{
			Browser:     cookies.BrowserInline,
			CookiesFile: path,
		},
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	if sessions[0].Browser != cookies.BrowserInline {
		t.Fatalf("Browser = %q, want %q", sessions[0].Browser, cookies.BrowserInline)
	}
	if !strings.HasPrefix(sessions[0].UserAgent, "Mozilla/5.0 ") {
		t.Fatalf("UserAgent = %q", sessions[0].UserAgent)
	}
}
