package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type fakeLoginResolver struct {
	login string
	err   error
}

func (f fakeLoginResolver) Login(string) (string, error) {
	return f.login, f.err
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

// blockingLoginResolver.Login blocks until gate is closed.
type blockingLoginResolver struct {
	gate  <-chan struct{}
	login string
}

func (r blockingLoginResolver) Login(string) (string, error) {
	<-r.gate
	return r.login, nil
}

// gatedProvider signals started (non-blocking) when Load runs.
type gatedProvider struct {
	started  chan<- struct{}
	sessions []browserprovider.BrowserSession
}

func (gatedProvider) Browser() cookies.Browser { return cookies.BrowserChromium }
func (gatedProvider) BackendName() string      { return "sweetcookie" }

func (p gatedProvider) Load(context.Context, string, cookies.Source) ([]browserprovider.BrowserSession, error) {
	select {
	case p.started <- struct{}{}:
	default:
	}
	return p.sessions, nil
}

// TestSessionResolver_CookieLoadingOverlapsLogin proves they run concurrently.
func TestSessionResolver_CookieLoadingOverlapsLogin(t *testing.T) {
	loginGate := make(chan struct{})
	releaseLogin := sync.OnceFunc(func() { close(loginGate) })
	defer releaseLogin() // safety net if the test fails before the release below

	loadStarted := make(chan struct{}, 1)
	providers := map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserChromium: gatedProvider{
			started: loadStarted,
			sessions: []browserprovider.BrowserSession{{
				Browser: cookies.BrowserChromium,
				Cookies: []*http.Cookie{{Name: "dotcom_user", Value: "sudosubin"}},
			}},
		},
	}

	sr := NewSessionResolver(
		blockingLoginResolver{gate: loginGate, login: "sudosubin"},
		NewCookieResolver(providers, false, nil),
	)

	resultCh := make(chan ResolvedCookies, 1)
	errCh := make(chan error, 1)
	go func() {
		resolved, err := sr.Resolve(t.Context(), "github.com", []cookies.Source{{Browser: cookies.BrowserChromium}})
		resultCh <- resolved
		errCh <- err
	}()

	select {
	case <-loadStarted: // ran while login is still blocked: genuinely concurrent
	case <-time.After(time.Second):
		t.Fatal("cookie loading never started before the timeout; Resolve appears to wait for login first")
	}

	releaseLogin()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		resolved := <-resultCh
		if resolved.Session.Browser != cookies.BrowserChromium {
			t.Fatalf("session browser = %q, want %q", resolved.Session.Browser, cookies.BrowserChromium)
		}
	case <-time.After(time.Second):
		t.Fatal("Resolve() did not return after login was released")
	}
}
