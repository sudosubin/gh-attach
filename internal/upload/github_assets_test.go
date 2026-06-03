package upload

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestPolicies_InjectsRefererAndUploadHeaders(t *testing.T) {
	const expectedReferer = "https://github.com/owner/repo/commit/abc123"
	const expectedFetchNonce = "nonce-123"
	const expectedClientVersion = "1.2.3"

	var receivedReferer string
	var receivedRequestedWith string
	var receivedVerifiedFetch string
	var receivedFetchNonce string
	var receivedClientVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/policies/assets" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		receivedReferer = r.Header.Get("Referer")
		receivedRequestedWith = r.Header.Get("X-Requested-With")
		receivedVerifiedFetch = r.Header.Get("GitHub-Verified-Fetch")
		receivedFetchNonce = r.Header.Get("X-Fetch-Nonce")
		receivedClientVersion = r.Header.Get("X-GitHub-Client-Version")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"https://uploads.github.test/upload","form":{},"header":{},"asset":{"id":1}}`))
	}))
	defer server.Close()

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       server.Client(),
	}
	refererPage := &RefererPage{
		URL:  expectedReferer,
		Body: []byte(`<meta name="fetch-nonce" content="nonce-123"><meta name="release" content="1.2.3">`),
	}

	_, err := u.requestPolicies(t.Context(), refererPage, "file.txt", 12, "text/plain")
	if err != nil {
		t.Fatalf("requestPolicies() error = %v", err)
	}
	if receivedReferer != expectedReferer {
		t.Fatalf("Referer header = %q, want %q", receivedReferer, expectedReferer)
	}
	if receivedRequestedWith != "XMLHttpRequest" {
		t.Fatalf("X-Requested-With = %q, want %q", receivedRequestedWith, "XMLHttpRequest")
	}
	if receivedVerifiedFetch != "true" {
		t.Fatalf("GitHub-Verified-Fetch = %q, want %q", receivedVerifiedFetch, "true")
	}
	if receivedFetchNonce != expectedFetchNonce {
		t.Fatalf("X-Fetch-Nonce = %q, want %q", receivedFetchNonce, expectedFetchNonce)
	}
	if receivedClientVersion != expectedClientVersion {
		t.Fatalf("X-GitHub-Client-Version = %q, want %q", receivedClientVersion, expectedClientVersion)
	}
}

func TestRequestPolicies_DoesNotInjectOptionalHeadersWhenMetaMissing(t *testing.T) {
	var receivedFetchNonce string
	var receivedClientVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedFetchNonce = r.Header.Get("X-Fetch-Nonce")
		receivedClientVersion = r.Header.Get("X-GitHub-Client-Version")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"https://uploads.github.test/upload","form":{},"header":{},"asset":{"id":1}}`))
	}))
	defer server.Close()

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       server.Client(),
	}
	refererPage := &RefererPage{
		URL:  "https://github.com/owner/repo/issues/new",
		Body: []byte{},
	}

	_, err := u.requestPolicies(t.Context(), refererPage, "file.txt", 12, "text/plain")
	if err != nil {
		t.Fatalf("requestPolicies() error = %v", err)
	}
	if receivedFetchNonce != "" {
		t.Fatalf("X-Fetch-Nonce = %q, want empty", receivedFetchNonce)
	}
	if receivedClientVersion != "" {
		t.Fatalf("X-GitHub-Client-Version = %q, want empty", receivedClientVersion)
	}
}

func TestCookieHeaderForURL_DeduplicatesKeysLastWins(t *testing.T) {
	t.Parallel()

	// Duplicate keys (e.g. kooky folding dFPI cookies into the default container) collapse last-wins instead of failing.
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

	// Host-only (github.com) and domain (.github.com) are distinct RFC 6265
	// scopes that can both live in one container, so this is not a leak.
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

func TestExtractRefererPageMetadata(t *testing.T) {
	html := `
		<meta name="csrf-token" content="csrf-1">
		<meta name="fetch-nonce" content="nonce-1">
		<meta name="release" content="release-1">
	`

	meta := extractRefererPageMetadata(html)
	if meta.AuthenticityToken != "csrf-1" {
		t.Fatalf("AuthenticityToken = %q, want %q", meta.AuthenticityToken, "csrf-1")
	}
	if meta.FetchNonce != "nonce-1" {
		t.Fatalf("FetchNonce = %q, want %q", meta.FetchNonce, "nonce-1")
	}
	if meta.GitHubClientVersion != "release-1" {
		t.Fatalf("GitHubClientVersion = %q, want %q", meta.GitHubClientVersion, "release-1")
	}
}
