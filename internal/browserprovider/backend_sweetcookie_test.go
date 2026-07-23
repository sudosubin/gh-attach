package browserprovider

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func writeInlineCookieFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "github-cookies.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

func TestSweetcookieBackendLoad_InlineJSONShapes(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"array":    `[{"name":"dotcom_user","value":"octocat","domain":".github.com","path":"/","secure":true}]`,
		"envelope": `{"cookies":[{"name":"dotcom_user","value":"octocat","domain":".github.com","path":"/","secure":true}]}`,
	}

	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sets, err := newSweetcookieBackend().Load(
				t.Context(),
				"github.com",
				cookies.Source{
					Browser:     cookies.BrowserInline,
					CookiesFile: writeInlineCookieFile(t, content),
				},
			)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if len(sets) != 1 || len(sets[0].Cookies) != 1 {
				t.Fatalf("sets = %#v", sets)
			}
			got := sets[0].Cookies[0]
			if got.Name != "dotcom_user" || got.Value != "octocat" {
				t.Fatalf("cookie = %#v", got)
			}
		})
	}
}

func TestSweetcookieBackendLoad_InlineFailures(t *testing.T) {
	t.Parallel()

	missingPath := filepath.Join(t.TempDir(), "missing.json")
	cases := []struct {
		name    string
		path    string
		content string
		want    string
	}{
		{
			name: "missing",
			path: missingPath,
			want: "read --cookies-file",
		},
		{
			name:    "empty",
			content: "   ",
			want:    "--cookies-file is empty",
		},
		{
			name:    "malformed",
			content: `{"cookies":[{"name":"user_session","value":"inline-secret"`,
			want:    "--cookies-file contains invalid JSON",
		},
		{
			name:    "wrong domain",
			content: `[{"name":"dotcom_user","value":"octocat","domain":"example.com","path":"/"}]`,
			want:    "no usable github.com cookies",
		},
		{
			name:    "expired",
			content: `[{"name":"dotcom_user","value":"octocat","domain":".github.com","path":"/","expires":1}]`,
			want:    "no usable github.com cookies",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := tc.path
			if path == "" {
				path = writeInlineCookieFile(t, tc.content)
			}
			_, err := newSweetcookieBackend().Load(
				t.Context(),
				"github.com",
				cookies.Source{
					Browser:     cookies.BrowserInline,
					CookiesFile: path,
				},
			)
			if err == nil {
				t.Fatalf("Load() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
			if strings.Contains(err.Error(), "inline-secret") {
				t.Fatalf("error leaked cookie value: %q", err.Error())
			}

			var safe interface{ SafeMessage() string }
			if !errors.As(err, &safe) {
				t.Fatalf("Load() error = %T, want SafeMessage()", err)
			}
			if !strings.Contains(safe.SafeMessage(), tc.want) {
				t.Fatalf("SafeMessage() = %q, want %q", safe.SafeMessage(), tc.want)
			}
		})
	}
}
