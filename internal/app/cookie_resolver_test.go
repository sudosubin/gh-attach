package app

import (
	"context"
	"errors"
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

// staticLogin builds a ghLogin thunk that always succeeds with login.
func staticLogin(login string) func() (string, error) {
	return func() (string, error) { return login, nil }
}

// trackingLogin builds a ghLogin thunk that records whether it was ever
// called, so tests can assert CookieResolver only pays for login resolution
// when a candidate session actually needs it.
func trackingLogin(login string, err error) (thunk func() (string, error), called func() bool) {
	var wasCalled bool
	thunk = func() (string, error) {
		wasCalled = true
		return login, err
	}
	called = func() bool { return wasCalled }
	return thunk, called
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
	resolved, err := resolver.Resolve(t.Context(), "github.com", staticLogin("sudosubin"), sources)
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

func TestCookieResolver_RejectsSessionsWithoutMatchingLogin(t *testing.T) {
	t.Parallel()

	tests := map[string][]*http.Cookie{
		"missing dotcom_user": {{Name: "other_cookie", Value: "value"}},
		"login mismatch":      {{Name: "dotcom_user", Value: "someone-else"}},
	}

	for name, sessionCookies := range tests {
		t.Run(name, func(t *testing.T) {
			providers := map[cookies.Browser]browserprovider.BrowserProvider{
				cookies.BrowserChromium: stubProvider{sessions: []browserprovider.BrowserSession{{
					Browser: cookies.BrowserChromium,
					Cookies: sessionCookies,
				}}},
			}
			resolver := NewCookieResolver(providers, false, nil)
			if _, err := resolver.Resolve(t.Context(), "github.com", staticLogin("sudosubin"), []cookies.Source{{Browser: cookies.BrowserChromium}}); err == nil {
				t.Fatal("Resolve() error = nil, want non-nil")
			}
		})
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
	_, err := resolver.Resolve(t.Context(), "github.com", staticLogin("sudosubin"), sources)
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
	resolved, err := resolver.Resolve(t.Context(), "github.com", staticLogin("sudosubin"), sources)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.ProviderName != "sweetcookie" {
		t.Fatalf("ProviderName = %q, want %q", resolved.ProviderName, "sweetcookie")
	}
}

func TestCookieResolver_PicksSessionMatchingLogin(t *testing.T) {
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
			backend:  "sweetcookie",
			sessions: []browserprovider.BrowserSession{otherSession, targetSession},
		},
	}

	resolver := NewCookieResolver(providers, false, nil)
	resolved, err := resolver.Resolve(t.Context(), "github.com", staticLogin("octocat"), sources)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Session.Profile != "default:sudosubin@example.com" {
		t.Fatalf("Session.Profile = %q, want %q", resolved.Session.Profile, "default:sudosubin@example.com")
	}
	// The resolver must return the whole session whose dotcom_user matches the
	// login, not an earlier non-matching one. (Container isolation itself is
	// covered by TestFinalizeContainerGroups_SplitsByContainerAndSortsDeterministically.)
	for _, c := range resolved.Session.Cookies {
		if c.Value == "sudo-session" || c.Value == "sudosubin" {
			t.Fatalf("resolved the wrong session: got a cookie from the non-matching one: %+v", c)
		}
	}
}

func TestCookieResolver_DoesNotCallGhLoginWithoutADotcomUserCandidate(t *testing.T) {
	t.Parallel()

	sources := []cookies.Source{{Browser: cookies.BrowserChromium}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			sessions: []browserprovider.BrowserSession{{
				Browser: cookies.BrowserChromium,
				Cookies: []*http.Cookie{{Name: "other_cookie", Value: "value"}},
			}},
		},
	}

	thunk, called := trackingLogin("sudosubin", nil)
	resolver := NewCookieResolver(providers, false, nil)
	if _, err := resolver.Resolve(t.Context(), "github.com", thunk, sources); err == nil {
		t.Fatal("Resolve() error = nil, want non-nil")
	}
	if called() {
		t.Fatal("ghLogin was called even though no candidate had a dotcom_user to compare")
	}
}

func TestCookieResolver_PropagatesGhLoginError(t *testing.T) {
	t.Parallel()

	sources := []cookies.Source{{Browser: cookies.BrowserChromium}}
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: stubProvider{
			backend: "sweetcookie",
			sessions: []browserprovider.BrowserSession{{
				Browser: cookies.BrowserChromium,
				Cookies: []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}},
			}},
		},
	}

	wantErr := errors.New("boom")
	thunk, called := trackingLogin("", wantErr)
	resolver := NewCookieResolver(providers, false, nil)
	_, err := resolver.Resolve(t.Context(), "github.com", thunk, sources)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Resolve() error = %v, want it to wrap %v", err, wantErr)
	}
	if !called() {
		t.Fatal("ghLogin was never called even though a candidate had a dotcom_user to compare")
	}
}
