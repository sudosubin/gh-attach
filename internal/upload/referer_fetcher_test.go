package upload

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/ghweb"
)

type stubLatestCommitResolver struct {
	sha string
	err error
}

func (r stubLatestCommitResolver) LatestCommitSHA(_ string, _ string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.sha, nil
}

func TestIssueNewPageFetcher_ReturnsNilOnStatusCodeError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	host := mustHost(t, server.URL)
	fetcher := NewIssueNewPageFetcher(host, "owner/repo")
	page, err := fetcher.Fetch(t.Context(), ghweb.NewClient(server.Client(), "", nil))
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if page != nil {
		t.Fatalf("Fetch() page = %#v, want nil", page)
	}
}

func TestResolveRefererPage_UsesFirstSuccessfulFetcher(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/owner/repo/issues/new":
			w.WriteHeader(http.StatusNotFound)
		case "/owner/repo/commit/abc123":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<meta name="csrf-token" content="commit-token">`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	host := mustHost(t, server.URL)
	uploader, err := NewUploader(host, 1, browserprovider.BrowserSession{Browser: cookies.BrowserChromium, UserAgent: "test-agent"}, server.Client())
	if err != nil {
		t.Fatalf("NewUploader() error = %v", err)
	}

	refererPage, err := uploader.ResolveRefererPage(
		t.Context(),
		[]RefererPageFetcher{
			NewIssueNewPageFetcher(host, "owner/repo"),
			NewLatestCommitPageFetcher(host, "owner", "repo", stubLatestCommitResolver{sha: "abc123"}),
		},
	)
	if err != nil {
		t.Fatalf("ResolveRefererPage() error = %v", err)
	}
	if refererPage.URL != "https://"+host+"/owner/repo/commit/abc123" {
		t.Fatalf("refererPage.URL = %q", refererPage.URL)
	}
	if string(refererPage.Body) != `<meta name="csrf-token" content="commit-token">` {
		t.Fatalf("refererPage.Body = %q", string(refererPage.Body))
	}
}

type stubRefererPageFetcher struct {
	calls int
	page  *RefererPage
	err   error
}

func (f *stubRefererPageFetcher) Fetch(_ context.Context, _ *ghweb.Client) (*RefererPage, error) {
	f.calls++
	return f.page, f.err
}

func TestResolveRefererPage_EarlyReturnsOnFirstSuccess(t *testing.T) {
	first := &stubRefererPageFetcher{page: &RefererPage{URL: "https://github.com/owner/repo/issues/new", Body: []byte("ok")}}
	second := &stubRefererPageFetcher{page: &RefererPage{URL: "https://github.com/owner/repo/commit/abc123", Body: []byte("ok")}}

	uploader, err := NewUploader("github.com", 1, browserprovider.BrowserSession{Browser: cookies.BrowserChromium, UserAgent: "test-agent"}, nil)
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
