package upload

import (
	"context"
	"net/http"
	"net/http/httptest"
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

	_, err := u.requestPolicies(context.Background(), refererPage, "file.txt", 12, "text/plain")
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

	_, err := u.requestPolicies(context.Background(), refererPage, "file.txt", 12, "text/plain")
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
