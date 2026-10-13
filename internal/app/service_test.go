package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/sudosubin/gh-attach/internal/github/attachments"
)

func TestServiceRun_UploadAuthentication(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		headers     http.Header
		wantCookies int
		wantErr     bool
		mixed       bool
	}{
		{name: "token succeeds", status: http.StatusCreated},
		{name: "mixed files", status: http.StatusCreated, wantCookies: 1, mixed: true},
		{name: "unauthorized falls back", status: http.StatusUnauthorized, wantCookies: 1},
		{name: "forbidden falls back", status: http.StatusForbidden, wantCookies: 1},
		{name: "not found falls back", status: http.StatusNotFound, wantCookies: 1},
		{name: "rate limited forbidden stops", status: http.StatusForbidden, headers: http.Header{"X-Ratelimit-Remaining": {"0"}}, wantErr: true},
		{name: "retry after stops", status: http.StatusForbidden, headers: http.Header{"Retry-After": {"60"}}, wantErr: true},
		{name: "too many requests stops", status: http.StatusTooManyRequests, wantErr: true},
		{name: "server error stops", status: http.StatusServiceUnavailable, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GH_TOKEN", "ghp_test_token")
			t.Setenv("GH_DEBUG", "")
			writeDefaultAttachConfig(t, "browsers:\n  - browser: firefox\n")
			files := []string{filepath.Join(t.TempDir(), "image.png")}
			if tt.mixed {
				files = append(files, filepath.Join(t.TempDir(), "report.pdf"))
			}
			for _, path := range files {
				if err := os.WriteFile(path, []byte("probe"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var tokenRequests, cookieRequests atomic.Int32
			svc := newBrowserTestService("github.com")
			svc.httpClient = &http.Client{Transport: uploadTestTransport(func(req *http.Request) (*http.Response, error) {
				status, body := http.StatusOK, ""
				headers := http.Header{"Content-Type": {"application/json"}}
				switch req.URL.Host + req.URL.Path {
				case "api.github.com/repos/owner/repo":
					body = `{"id":123}`
				case "uploads.github.com/user-attachments/assets":
					tokenRequests.Add(1)
					if req.Header.Get("Authorization") != "token ghp_test_token" || req.URL.Query().Get("repository_id") != "123" {
						t.Errorf("token authentication/repository = %v, %s", req.Header, req.URL)
					}
					status, body = tt.status, `{"url":"https://github.com/user-attachments/assets/token","message":"denied"}`
					for key, values := range tt.headers {
						headers[key] = values
					}
				case "github.com/owner/repo/issues/new":
					cookieRequests.Add(1)
					if !strings.Contains(req.Header.Get("Cookie"), "user_session=browser-token") || req.Header.Get("Authorization") != "" {
						t.Errorf("browser authentication = %v", req.Header)
					}
					body = `<meta name="fetch-nonce" content="nonce"><meta name="octolytics-dimension-repository_id" content="123">`
				case "github.com/upload/policies/assets":
					body = `{"upload_url":"https://storage.example.com/upload"}`
				case "storage.example.com/upload":
					body = `{"href":"https://github.com/user-attachments/assets/cookie"}`
				default:
					return nil, fmt.Errorf("unexpected upload request: %s", req.URL)
				}
				return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})}
			assets, err := svc.Run(t.Context(), Request{FilePaths: files, Repo: "github.com/owner/repo"})
			if tokenRequests.Load() != 1 || cookieRequests.Load() != int32(tt.wantCookies) {
				t.Errorf("requests = token %d, cookie %d; want 1, %d", tokenRequests.Load(), cookieRequests.Load(), tt.wantCookies)
			}
			if tt.wantErr {
				httpErr, ok := errors.AsType[*api.HTTPError](err)
				if !ok || httpErr.StatusCode != tt.status || len(assets) != 0 {
					t.Fatalf("Run() = %v, %v; want no assets and HTTP %d", assets, err, tt.status)
				}
				return
			}
			if err != nil || len(assets) != len(files) {
				t.Fatalf("Run() = %v, %v; want %d assets", assets, err, len(files))
			}
			for i, asset := range assets {
				wantHref := "https://github.com/user-attachments/assets/token"
				if tt.wantCookies != 0 && (!tt.mixed || i == 1) {
					wantHref = "https://github.com/user-attachments/assets/cookie"
				}
				if asset.Href != wantHref || asset.Name != filepath.Base(files[i]) {
					t.Errorf("asset %d = %v, want %s at %s", i, asset, filepath.Base(files[i]), wantHref)
				}
			}
		})
	}
}

type uploadTestTransport func(*http.Request) (*http.Response, error)

func (f uploadTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUploadFiles_ContinuesAfterError(t *testing.T) {
	assets, err := uploadFiles([]string{"one.png", "bad.png", "two.png"}, func(path string) (attachments.Asset, error) {
		if path == "bad.png" {
			return attachments.Asset{}, errors.New("rejected")
		}
		return attachments.Asset{Href: "https://example.com/" + path}, nil
	})

	if len(assets) != 2 {
		t.Fatalf("len(assets) = %d, want 2", len(assets))
	}
	got := []string{assets[0].Href, assets[1].Href}
	want := []string{"https://example.com/one.png", "https://example.com/two.png"}
	if !slices.Equal(got, want) {
		t.Fatalf("assets = %v, want %v", got, want)
	}
	if err == nil || !strings.Contains(err.Error(), "bad.png: rejected") {
		t.Fatalf("error = %v, want file-scoped rejection", err)
	}
}

func TestUploadFiles_UploadsConcurrently(t *testing.T) {
	started := make(chan struct{}, maxConcurrentUploads)
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = uploadFiles([]string{"one.png", "two.png"}, func(string) (attachments.Asset, error) {
			started <- struct{}{}
			<-release
			return attachments.Asset{}, nil
		})
	}()

	for range maxConcurrentUploads {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("uploads did not start concurrently")
		}
	}
	close(release)
	<-done
}
