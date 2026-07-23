package ghweb

import (
	"net/http"
	"strings"
	"testing"
)

func TestCookieHeaderForURL_DeduplicatesKeysLastWins(t *testing.T) {
	t.Parallel()

	// Duplicate keys (e.g. dFPI cookies folded into the default container) collapse last-wins instead of failing.
	dupes := []*http.Cookie{
		{Name: "user_session", Value: "stale", Domain: "github.com", Path: "/"},
		{Name: "user_session", Value: "current", Domain: "github.com", Path: "/"},
	}

	got, err := cookieHeaderForURL(dupes, "https://github.com/")
	if err != nil {
		t.Fatalf("cookieHeaderForURL() error = %v, want nil", err)
	}
	if got != "user_session=current" {
		t.Fatalf("header = %q, want user_session=current", got)
	}
}

func TestCookieHeaderForURL_AcceptsHostOnlyAndDomainScopes(t *testing.T) {
	t.Parallel()

	// Host-only and domain-scoped cookies are distinct RFC 6265 scopes, not a leak.
	in := []*http.Cookie{
		{Name: "user_session", Value: "host-only", Domain: "github.com", Path: "/"},
		{Name: "user_session", Value: "domain", Domain: ".github.com", Path: "/"},
	}

	got, err := cookieHeaderForURL(in, "https://github.com/")
	if err != nil {
		t.Fatalf("cookieHeaderForURL() error = %v, want nil", err)
	}
	if !strings.Contains(got, "user_session=") {
		t.Fatalf("header = %q, want a user_session cookie", got)
	}
}

func TestCookieHeaderForURL_AcceptsDistinctKeys(t *testing.T) {
	t.Parallel()

	in := []*http.Cookie{
		{Name: "dotcom_user", Value: "octocat", Domain: "github.com", Path: "/"},
		{Name: "user_session", Value: "abc", Domain: "github.com", Path: "/"},
	}

	got, err := cookieHeaderForURL(in, "https://github.com/")
	if err != nil {
		t.Fatalf("cookieHeaderForURL() error = %v", err)
	}
	if !strings.Contains(got, "dotcom_user=octocat") || !strings.Contains(got, "user_session=abc") {
		t.Fatalf("header = %q, missing expected cookies", got)
	}
}

func TestCookieHeaderForURL_ExcludesSiblingSubdomain(t *testing.T) {
	t.Parallel()

	// Must not leak to a sibling subdomain, even though cookiejar would allow it via suffix matching.
	in := []*http.Cookie{
		{Name: "user_session", Value: "abc", Domain: "github.test", Path: "/"},
	}

	got, err := cookieHeaderForURL(in, "https://media.github.test/user/1/files")
	if err != nil {
		t.Fatalf("cookieHeaderForURL() error = %v, want nil", err)
	}
	if got != "" {
		t.Fatalf("header = %q, want empty (cookie must not leak to sibling subdomain)", got)
	}
}
