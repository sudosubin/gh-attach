package attach

import (
	"context"
	"net/http"
	"testing"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type stubProvider struct {
	backend string
	session browserprovider.BrowserSession
	err     error
}

func (p stubProvider) Browser() cookies.Browser { return p.session.Browser }
func (p stubProvider) BackendName() string      { return p.backend }
func (p stubProvider) Load(_ context.Context, _ string, _ cookies.Source) (browserprovider.BrowserSession, error) {
	if p.err != nil {
		return browserprovider.BrowserSession{}, p.err
	}
	return p.session, nil
}

func TestCookieResolver_MatchesAnyDotcomUserValue(t *testing.T) {
	t.Parallel()

	sources := []cookies.Source{{Browser: cookies.BrowserChromium}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			session: browserprovider.BrowserSession{
				Browser: cookies.BrowserChromium,
				Cookies: []*http.Cookie{
					{Name: "dotcom_user", Value: "other-user"},
					{Name: "dotcom_user", Value: "sudosubin"},
				},
				UserAgent: "ua",
			},
		},
	}

	resolver := NewCookieResolver(providers, false, nil)
	resolved, err := resolver.Resolve(context.Background(), "github.com", "sudosubin", sources)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Source.Browser != cookies.BrowserChromium {
		t.Fatalf("selected browser = %q, want %q", resolved.Source.Browser, cookies.BrowserChromium)
	}
	if resolved.Session.Browser != cookies.BrowserChromium {
		t.Fatalf("session browser = %q, want %q", resolved.Session.Browser, cookies.BrowserChromium)
	}
}

func TestCookieResolver_SkipsOnMissingDotcomUser(t *testing.T) {
	t.Parallel()

	sources := []cookies.Source{{Browser: cookies.BrowserChromium}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			session: browserprovider.BrowserSession{
				Browser:   cookies.BrowserChromium,
				Cookies:   []*http.Cookie{{Name: "other_cookie", Value: "value"}},
				UserAgent: "ua",
			},
		},
	}

	resolver := NewCookieResolver(providers, false, nil)
	_, err := resolver.Resolve(context.Background(), "github.com", "sudosubin", sources)
	if err == nil {
		t.Fatalf("Resolve() error = nil, want non-nil")
	}
}

func TestCookieResolver_SkipsOnLoginMismatch(t *testing.T) {
	t.Parallel()

	sources := []cookies.Source{{Browser: cookies.BrowserChromium}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			session: browserprovider.BrowserSession{
				Browser:   cookies.BrowserChromium,
				Cookies:   []*http.Cookie{{Name: "dotcom_user", Value: "someone-else"}},
				UserAgent: "ua",
			},
		},
	}

	resolver := NewCookieResolver(providers, false, nil)
	_, err := resolver.Resolve(context.Background(), "github.com", "sudosubin", sources)
	if err == nil {
		t.Fatalf("Resolve() error = nil, want non-nil")
	}
}

func TestCookieResolver_SkipsMissingProvider(t *testing.T) {
	t.Parallel()

	// firefox source, but no firefox in providers → exhausts all, returns error
	sources := []cookies.Source{{Browser: cookies.BrowserFirefox}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			session: browserprovider.BrowserSession{Browser: cookies.BrowserChromium},
		},
	}

	resolver := NewCookieResolver(providers, false, nil)
	_, err := resolver.Resolve(context.Background(), "github.com", "sudosubin", sources)
	if err == nil {
		t.Fatalf("Resolve() error = nil, want non-nil")
	}
}

func TestCookieResolver_LoginMatchIsCaseInsensitive(t *testing.T) {
	t.Parallel()

	sources := []cookies.Source{{Browser: cookies.BrowserChromium}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			session: browserprovider.BrowserSession{
				Browser:   cookies.BrowserChromium,
				Cookies:   []*http.Cookie{{Name: "dotcom_user", Value: "SudoSubin"}},
				UserAgent: "ua",
			},
		},
	}

	resolver := NewCookieResolver(providers, false, nil)
	resolved, err := resolver.Resolve(context.Background(), "github.com", "sudosubin", sources)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.ProviderName != "sweetcookie" {
		t.Fatalf("ProviderName = %q, want %q", resolved.ProviderName, "sweetcookie")
	}
}
