package attach

import (
	"context"
	"net/http"
	"testing"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type stubProvider struct {
	backend  string
	sessions []browserprovider.BrowserSession
	err      error
}

func (p stubProvider) Browser() cookies.Browser {
	if len(p.sessions) == 0 {
		return ""
	}
	return p.sessions[0].Browser
}

func (p stubProvider) BackendName() string { return p.backend }

func (p stubProvider) Load(_ context.Context, _ string, _ cookies.Source) ([]browserprovider.BrowserSession, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.sessions, nil
}

func TestCookieResolver_MatchesAnyDotcomUserValue(t *testing.T) {
	t.Parallel()

	sources := []cookies.Source{{Browser: cookies.BrowserChromium}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			sessions: []browserprovider.BrowserSession{{
				Browser: cookies.BrowserChromium,
				Profile: "Default",
				Cookies: []*http.Cookie{
					{Name: "dotcom_user", Value: "other-user"},
					{Name: "dotcom_user", Value: "sudosubin"},
				},
				UserAgent: "ua",
			}},
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
			sessions: []browserprovider.BrowserSession{{
				Browser:   cookies.BrowserChromium,
				Cookies:   []*http.Cookie{{Name: "other_cookie", Value: "value"}},
				UserAgent: "ua",
			}},
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
			sessions: []browserprovider.BrowserSession{{
				Browser:   cookies.BrowserChromium,
				Cookies:   []*http.Cookie{{Name: "dotcom_user", Value: "someone-else"}},
				UserAgent: "ua",
			}},
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
			backend:  "sweetcookie",
			sessions: []browserprovider.BrowserSession{{Browser: cookies.BrowserChromium}},
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
			sessions: []browserprovider.BrowserSession{{
				Browser:   cookies.BrowserChromium,
				Cookies:   []*http.Cookie{{Name: "dotcom_user", Value: "SudoSubin"}},
				UserAgent: "ua",
			}},
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

func TestCookieResolver_PicksMatchingContainerSessionAndIsolatesCookies(t *testing.T) {
	t.Parallel()

	otherSession := browserprovider.BrowserSession{
		Browser: cookies.BrowserFirefox,
		Profile: "default:sudosubin@gmail.com",
		Cookies: []*http.Cookie{
			{Name: "dotcom_user", Value: "sudosubin"},
			{Name: "user_session", Value: "sudo-session"},
		},
		UserAgent: "ua",
	}
	targetSession := browserprovider.BrowserSession{
		Browser: cookies.BrowserFirefox,
		Profile: "default:sudosubin@example.com",
		Cookies: []*http.Cookie{
			{Name: "dotcom_user", Value: "octocat"},
			{Name: "user_session", Value: "octocat-session"},
		},
		UserAgent: "ua",
	}

	sources := []cookies.Source{{Browser: cookies.BrowserFirefox}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserFirefox: stubProvider{
			backend:  "kooky",
			sessions: []browserprovider.BrowserSession{otherSession, targetSession},
		},
	}

	resolver := NewCookieResolver(providers, false, nil)
	resolved, err := resolver.Resolve(context.Background(), "github.com", "octocat", sources)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Session.Profile != "default:sudosubin@example.com" {
		t.Fatalf("Session.Profile = %q, want %q", resolved.Session.Profile, "default:sudosubin@example.com")
	}
	// Cookies from the other session must not leak into the resolved one.
	for _, c := range resolved.Session.Cookies {
		if c.Value == "sudo-session" || c.Value == "sudosubin" {
			t.Fatalf("resolved session leaked cookie from other container: %+v", c)
		}
	}
}
