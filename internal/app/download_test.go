package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestParseAttachmentURL(t *testing.T) {
	// given
	t.Setenv("GH_HOST", "ghe.example.com")

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name: "cloud asset",
			url:  "https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "cloud file",
			url:  "https://github.com/user-attachments/files/123/report.pdf",
		},
		{
			name: "ghes subdomain isolation",
			url:  "https://media.ghe.example.com/user-attachments/assets/id",
		},
		{
			name:    "unconfigured port on known host",
			url:     "https://ghe.example.com:8443/user-attachments/assets/id",
			wantErr: true,
		},
		{
			name:    "query string",
			url:     "https://github.com/user-attachments/assets/id?download=1",
			wantErr: true,
		},
		{
			name:    "trailing slash",
			url:     "https://github.com/user-attachments/files/123/report.pdf/",
			wantErr: true,
		},
		{
			name:    "unknown host",
			url:     "https://example.com/user-attachments/assets/id",
			wantErr: true,
		},
		{
			name:    "non attachment path",
			url:     "https://github.com/owner/repo/issues/1",
			wantErr: true,
		},
		{
			name:    "embedded credentials",
			url:     "https://user:secret@github.com/user-attachments/assets/id",
			wantErr: true,
		},
		{
			name:    "insecure URL",
			url:     "http://github.com/user-attachments/assets/id",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// when
			_, _, err := parseAttachmentURL(test.url)

			// then
			if (err != nil) != test.wantErr {
				t.Fatalf("parseAttachmentURL() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestServiceDownload_ExplicitSessionTokenPrecedesBearer(t *testing.T) {
	// given
	var authorization, cookie string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		cookie = r.Header.Get("Cookie")
		_, _ = w.Write([]byte("image"))
	}))
	defer server.Close()

	t.Setenv("GH_HOST", mustServerURL(t, server.URL).Host)
	t.Setenv("GH_ENTERPRISE_TOKEN", "gh-token")

	svc := NewService(io.Discard)
	svc.httpClient = server.Client()

	// when
	body, err := svc.Download(t.Context(), DownloadRequest{
		URL:          server.URL + "/user-attachments/assets/id",
		SessionToken: "session-token",
	})
	// then
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = body.Close() }()
	if _, err := io.ReadAll(body); err != nil {
		t.Fatal(err)
	}

	if authorization != "" {
		t.Fatalf("Authorization = %q, want empty", authorization)
	}
	if !strings.Contains(cookie, "user_session=session-token") {
		t.Fatalf("Cookie = %q, want session token", cookie)
	}
}

func TestServiceDownload_BearerFallsBackToBrowserCookies(t *testing.T) {
	// given
	var bearerRequests, cookieRequests int
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Header.Get("Authorization") == "Bearer gh-token":
			bearerRequests++
			http.Error(w, "Not Found", http.StatusNotFound)
		case strings.Contains(r.Header.Get("Cookie"), "user_session=browser-token"):
			cookieRequests++
			_, _ = w.Write([]byte("image"))
		default:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	serverURL := mustServerURL(t, server.URL)
	configureDownloadTestAuth(t, serverURL.Host)

	svc := NewService(io.Discard)
	svc.httpClient = server.Client()
	svc.loginResolver = fakeLoginResolver{login: "octocat"}
	svc.providers = map[cookies.Browser]browserprovider.BrowserProvider{
		cookies.BrowserFirefox: stubProvider{
			backend: "sweetcookie",
			sessions: []browserprovider.BrowserSession{{
				Browser: cookies.BrowserFirefox,
				Profile: "default",
				Cookies: []*http.Cookie{
					{Name: "dotcom_user", Value: "octocat", Domain: serverURL.Hostname(), Path: "/", Secure: true},
					{Name: "user_session", Value: "browser-token", Domain: serverURL.Hostname(), Path: "/", Secure: true},
				},
				UserAgent: "firefox",
			}},
		},
	}

	// when
	body, err := svc.Download(t.Context(), DownloadRequest{
		URL:     server.URL + "/user-attachments/assets/id",
		Browser: "firefox",
	})
	// then
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = body.Close() }()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "image" {
		t.Fatalf("body = %q", got)
	}
	if bearerRequests != 0 {
		t.Fatalf("bearer requests = %d, want 0 for explicit browser", bearerRequests)
	}
	if cookieRequests != 1 {
		t.Fatalf("cookie requests = %d, want 1", cookieRequests)
	}

	// when
	fallback, err := svc.Download(t.Context(), DownloadRequest{
		URL: server.URL + "/user-attachments/assets/id",
	})
	// then
	if err != nil {
		t.Fatalf("Download() with fallback error = %v", err)
	}
	defer func() { _ = fallback.Close() }()
	if _, err := io.ReadAll(fallback); err != nil {
		t.Fatal(err)
	}
	if bearerRequests != 1 || cookieRequests != 2 {
		t.Fatalf("requests = bearer %d, cookie %d, want 1 and 2", bearerRequests, cookieRequests)
	}
}

func TestServiceDownload_StripsAuthenticationAfterRedirect(t *testing.T) {
	// given
	var redirectedAuthorization, redirectedCookie string
	assetServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedAuthorization = r.Header.Get("Authorization")
		redirectedCookie = r.Header.Get("Cookie")
		_, _ = w.Write([]byte("image"))
	}))
	defer assetServer.Close()

	entryServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer gh-token" {
			t.Errorf("Authorization = %q, want Bearer token", r.Header.Get("Authorization"))
		}
		http.Redirect(w, r, assetServer.URL+"/image", http.StatusFound)
	}))
	defer entryServer.Close()

	t.Setenv("GH_HOST", mustServerURL(t, entryServer.URL).Host)
	t.Setenv("GH_ENTERPRISE_TOKEN", "gh-token")

	client := entryServer.Client()
	client.Transport = &redirectTransport{
		entryHost: entryServer.Listener.Addr().String(),
		assetHost: assetServer.Listener.Addr().String(),
		entryTLS:  entryServer.Client().Transport,
		assetTLS:  assetServer.Client().Transport,
	}

	svc := NewService(io.Discard)
	svc.httpClient = client

	// when
	body, err := svc.Download(t.Context(), DownloadRequest{
		URL: entryServer.URL + "/user-attachments/assets/id",
	})
	// then
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer func() { _ = body.Close() }()
	if _, err := io.ReadAll(body); err != nil {
		t.Fatal(err)
	}
	if redirectedAuthorization != "" || redirectedCookie != "" {
		t.Fatalf("redirected credentials = authorization %q, cookie %q", redirectedAuthorization, redirectedCookie)
	}
}

type redirectTransport struct {
	entryHost string
	assetHost string
	entryTLS  http.RoundTripper
	assetTLS  http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == t.entryHost {
		return t.entryTLS.RoundTrip(req)
	}
	if req.URL.Host == t.assetHost {
		return t.assetTLS.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func mustServerURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func configureDownloadTestAuth(t *testing.T, host string) {
	t.Helper()
	t.Setenv("GH_HOST", host)
	t.Setenv("GH_ENTERPRISE_TOKEN", "gh-token")

	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	attachDir := filepath.Join(xdgDir, "gh")
	if err := os.MkdirAll(attachDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attachDir, "attach.yml"), []byte("browsers:\n  - browser: firefox\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}
