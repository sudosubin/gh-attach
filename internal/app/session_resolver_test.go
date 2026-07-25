package app

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type fakeAPILogin struct {
	login string
	err   error
	calls int
}

func (f *fakeAPILogin) CurrentLogin() (string, error) {
	f.calls++
	return f.login, f.err
}

type fakeLoginResolver struct {
	login string
	err   error
}

func (f fakeLoginResolver) Login(string) (string, error) {
	return f.login, f.err
}

func TestConfigLoginResolver_FallsBackToAPIWhenNoLocalToken(t *testing.T) {
	// No env token and no config file -> loginFromLocalToken declines, forcing the REST fallback.
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_ENTERPRISE_TOKEN", "")
	t.Setenv("GITHUB_ENTERPRISE_TOKEN", "")

	api := &fakeAPILogin{login: "api-user"}
	got, err := NewConfigLoginResolver(api).Login("nonexistent.invalid")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if got != "api-user" {
		t.Fatalf("Login() = %q, want %q", got, "api-user")
	}
	if api.calls != 1 {
		t.Fatalf("api.calls = %d, want 1", api.calls)
	}
}

func TestSessionResolver_UsesLoginToMatchCookies(t *testing.T) {
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			sessions: []browserprovider.BrowserSession{{
				Browser:   cookies.BrowserChromium,
				Profile:   "Default",
				Cookies:   []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}},
				UserAgent: "ua",
			}},
		},
	}

	sr := NewSessionResolver(
		fakeLoginResolver{login: "sudosubin"},
		NewCookieResolver(providers, false, nil),
	)

	resolved, err := sr.Resolve(t.Context(), "github.com", []cookies.Source{{Browser: cookies.BrowserChromium}})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Session.Browser != cookies.BrowserChromium {
		t.Fatalf("session browser = %q, want %q", resolved.Session.Browser, cookies.BrowserChromium)
	}
	if resolved.ProviderName != "sweetcookie" {
		t.Fatalf("provider = %q, want %q", resolved.ProviderName, "sweetcookie")
	}
}

func TestSessionResolver_PropagatesLoginError(t *testing.T) {
	sr := NewSessionResolver(
		fakeLoginResolver{err: fmt.Errorf("boom")},
		NewCookieResolver(nil, false, nil),
	)

	if _, err := sr.Resolve(t.Context(), "github.com", nil); err == nil {
		t.Fatalf("Resolve() error = nil, want non-nil")
	}
}
