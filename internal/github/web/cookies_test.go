package web

import (
	"net/http"
	"net/url"
	"testing"
)

func TestScopedCookieJar_DeduplicatesKeysLastWins(t *testing.T) {
	t.Parallel()

	jar := newScopedCookieJar([]*http.Cookie{
		{Name: "user_session", Value: "stale", Domain: "github.com", Path: "/"},
		{Name: "user_session", Value: "current", Domain: "github.com", Path: "/"},
	})
	got := jar.Cookies(mustURL(t, "https://github.com/"))
	if len(got) != 1 || got[0].Value != "current" {
		t.Fatalf("cookies = %+v, want current user_session", got)
	}
}

func TestScopedCookieJar_ExcludesSiblingSubdomain(t *testing.T) {
	t.Parallel()

	jar := newScopedCookieJar([]*http.Cookie{
		{Name: "user_session", Value: "abc", Domain: "github.test", Path: "/"},
	})
	if got := jar.Cookies(mustURL(t, "https://media.github.test/user/1/files")); len(got) != 0 {
		t.Fatalf("cookies = %+v, want none", got)
	}
}

func mustURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	return u
}
