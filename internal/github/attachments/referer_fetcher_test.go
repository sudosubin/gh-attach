package attachments

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/web"
)

func TestIssueNewPageFetcher_ReturnsNilOnStatusCodeError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	host := mustHost(t, server.URL)
	fetcher := NewIssueNewPageFetcher(host, "owner/repo")
	page, err := fetcher.Fetch(t.Context(), web.NewClient(server.Client(), "", nil))
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if page != nil {
		t.Fatalf("Fetch() page = %#v, want nil", page)
	}
}

func TestResolveRefererPage_UsesFirstSuccessfulFetcher(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /owner/repo/issues/new", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /owner/repo/commits/HEAD", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<meta name="csrf-token" content="commit-token"><meta name="octolytics-dimension-repository_id" content="777">`))
	})

	server := httptest.NewTLSServer(mux)
	defer server.Close()

	host := mustHost(t, server.URL)
	uploader, err := NewUploader(host, browserprovider.BrowserSession{Browser: cookies.BrowserChromium, UserAgent: "test-agent"}, server.Client())
	if err != nil {
		t.Fatalf("NewUploader() error = %v", err)
	}

	refererPage, err := uploader.ResolveRefererPage(
		t.Context(),
		[]RefererPageFetcher{
			NewIssueNewPageFetcher(host, "owner/repo"),
			NewCommitsHeadPageFetcher(host, "owner/repo"),
		},
	)
	if err != nil {
		t.Fatalf("ResolveRefererPage() error = %v", err)
	}
	if refererPage.URL != "https://"+host+"/owner/repo/commits/HEAD" {
		t.Fatalf("refererPage.URL = %q", refererPage.URL)
	}
	// Parsed at fetch time: CSRF via the shared parser, repository_id via the octolytics pattern.
	if refererPage.Meta.AuthenticityToken != "commit-token" {
		t.Fatalf("refererPage.Meta.AuthenticityToken = %q, want %q", refererPage.Meta.AuthenticityToken, "commit-token")
	}
	if refererPage.Meta.RepositoryID != 777 {
		t.Fatalf("refererPage.Meta.RepositoryID = %d, want %d", refererPage.Meta.RepositoryID, 777)
	}
}

type stubRefererPageFetcher struct {
	calls int
	page  *RefererPage
	err   error
}

func (f *stubRefererPageFetcher) Fetch(_ context.Context, _ *web.Client) (*RefererPage, error) {
	f.calls++
	return f.page, f.err
}

func TestResolveRefererPage_EarlyReturnsOnFirstSuccess(t *testing.T) {
	first := &stubRefererPageFetcher{page: &RefererPage{URL: "https://github.com/owner/repo/issues/new"}}
	second := &stubRefererPageFetcher{page: &RefererPage{URL: "https://github.com/owner/repo/commit/abc123"}}

	uploader, err := NewUploader("github.com", browserprovider.BrowserSession{Browser: cookies.BrowserChromium, UserAgent: "test-agent"}, nil)
	if err != nil {
		t.Fatalf("NewUploader() error = %v", err)
	}

	resolved, err := uploader.ResolveRefererPage(
		t.Context(),
		[]RefererPageFetcher{first, second},
	)
	if err != nil {
		t.Fatalf("ResolveRefererPage() error = %v", err)
	}
	if resolved.URL != first.page.URL {
		t.Fatalf("resolved.URL = %q, want %q", resolved.URL, first.page.URL)
	}
	if first.calls != 1 || second.calls != 0 {
		t.Fatalf("calls first=%d second=%d, want first=1 second=0", first.calls, second.calls)
	}
}

func mustHost(t *testing.T, rawURL string) string {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	return u.Host
}

func mustOrigin(t *testing.T, rawURL string) string {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	return u.Scheme + "://" + u.Host
}

func TestParseCSRFMetadata(t *testing.T) {
	html := `
		<meta name="csrf-token" content="csrf-1">
		<meta name="fetch-nonce" content="nonce-1">
		<meta name="release" content="release-1">
	`

	meta := parseCSRFMetadata(html)
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

func TestExtractRepositoryID(t *testing.T) {
	tests := []struct {
		name string
		html string
		want int64
	}{
		{
			name: "react databaseId (issues/new on github.com)",
			html: `<script>{"repository":{"id":"R_kgABC","databaseId":261246700,"name":"nixos-config"}}</script>`,
			want: 261246700,
		},
		{
			name: "octolytics meta (commit on github.com)",
			html: `<meta name="octolytics-dimension-repository_id" content="261246700">`,
			want: 261246700,
		},
		{
			name: "deferred-side-panel data-url (issues/new on GHES)",
			html: `<deferred-side-panel data-url="/_side-panels/user?repository_id=416">`,
			want: 416,
		},
		{
			name: "databaseId takes priority over octolytics when both present",
			html: `<script>{"repository":{"id":"R_kgABC","databaseId":100}}</script><meta name="octolytics-dimension-repository_id" content="200">`,
			want: 100,
		},
		{
			name: "octolytics takes priority over data-url when both present",
			html: `<meta name="octolytics-dimension-repository_id" content="100"><deferred-side-panel data-url="/x?repository_id=200">`,
			want: 100,
		},
		{
			name: "no repository id present",
			html: `<html><head><title>nothing here</title></head></html>`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRepositoryID(tt.html); got != tt.want {
				t.Fatalf("extractRepositoryID() = %d, want %d", got, tt.want)
			}
		})
	}
}
