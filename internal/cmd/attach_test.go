package cmd

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
func (p stubProvider) Load(ctx context.Context, host string, source cookies.Source) (browserprovider.BrowserSession, error) {
	if p.err != nil {
		return browserprovider.BrowserSession{}, p.err
	}
	return p.session, nil
}

func TestResolveCookies_MatchesAnyDotcomUserValue(t *testing.T) {
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

	session, selectedSource, _, err := resolveCookies(context.Background(), "github.com", "sudosubin", sources, providers, false)
	if err != nil {
		t.Fatalf("resolveCookies() error = %v", err)
	}
	if selectedSource.Browser != cookies.BrowserChromium {
		t.Fatalf("selected browser = %q, want %q", selectedSource.Browser, cookies.BrowserChromium)
	}
	if session.Browser != cookies.BrowserChromium {
		t.Fatalf("session browser = %q, want %q", session.Browser, cookies.BrowserChromium)
	}
}

func TestCookieValuesByNameForHost_ExcludesOtherSubdomains(t *testing.T) {
	t.Parallel()

	in := []*http.Cookie{
		{Name: "dotcom_user", Value: "sudosubin", Domain: "gist.github.com", Path: "/"},
		{Name: "dotcom_user", Value: "other-user", Domain: "github.com", Path: "/"},
	}

	got := cookieValuesByNameForHost(in, "dotcom_user", "github.com")
	if len(got) != 1 || got[0] != "other-user" {
		t.Fatalf("cookieValuesByNameForHost() = %#v, want [other-user]", got)
	}
}

func TestApplyDefaultProfile_ChromiumFamily(t *testing.T) {
	t.Parallel()

	for _, b := range []cookies.Browser{
		cookies.BrowserChrome,
		cookies.BrowserChromium,
		cookies.BrowserEdge,
		cookies.BrowserBrave,
		cookies.BrowserVivaldi,
		cookies.BrowserOpera,
	} {
		got := applyDefaultProfile(cookies.Source{Browser: b})
		if got.Profile != "Default" {
			t.Fatalf("browser=%s profile=%q, want %q", b, got.Profile, "Default")
		}
	}
}

func TestApplyDefaultProfile_DoesNotOverrideExplicitProfileOrPath(t *testing.T) {
	t.Parallel()

	withProfile := applyDefaultProfile(cookies.Source{Browser: cookies.BrowserChromium, Profile: "Work"})
	if withProfile.Profile != "Work" {
		t.Fatalf("profile=%q, want %q", withProfile.Profile, "Work")
	}

	withPath := applyDefaultProfile(cookies.Source{Browser: cookies.BrowserChromium, CookieStorePath: "/tmp/Cookies"})
	if withPath.Profile != "" {
		t.Fatalf("profile=%q, want empty", withPath.Profile)
	}

	firefox := applyDefaultProfile(cookies.Source{Browser: cookies.BrowserFirefox})
	if firefox.Profile != "" {
		t.Fatalf("firefox profile=%q, want empty", firefox.Profile)
	}
}
