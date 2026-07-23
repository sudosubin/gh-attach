package browserprovider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChromiumMajorVersion_LastVersionWins(t *testing.T) {
	t.Parallel()

	// given
	udd := t.TempDir()
	store := filepath.Join(udd, "Default", "Cookies")
	writeFile(t, store, "")
	writeFile(t, filepath.Join(udd, "Last Version"), "150.0.7871.128\n")
	writeFile(t, filepath.Join(udd, "Default", "Preferences"), `{"profile":{"created_by_version":"131.0.0.0"}}`)

	// when
	got := (chromiumFamily{}).version(store)

	// then
	if got != "150" {
		t.Fatalf("major = %q, want 150", got)
	}
}

func TestChromiumMajorVersion_NetworkSubdirLayout(t *testing.T) {
	t.Parallel()

	// given
	udd := t.TempDir()
	store := filepath.Join(udd, "Default", "Network", "Cookies")
	writeFile(t, store, "")
	writeFile(t, filepath.Join(udd, "Last Version"), "150.0.7871.128")

	// when
	got := (chromiumFamily{}).version(store)

	// then
	if got != "150" {
		t.Fatalf("major = %q, want 150 (Network layout)", got)
	}
}

func TestChromiumMajorVersion_PreferencesFallback(t *testing.T) {
	t.Parallel()

	// given
	udd := t.TempDir()
	store := filepath.Join(udd, "Default", "Cookies")
	writeFile(t, store, "")
	writeFile(t, filepath.Join(udd, "Default", "Preferences"), `{"profile":{"created_by_version":"133.0.5932.60"}}`)

	// when
	got := (chromiumFamily{}).version(store)

	// then
	if got != "133" {
		t.Fatalf("major = %q, want 133", got)
	}
}

func TestChromiumMajorVersion_NoFilesReturnsEmpty(t *testing.T) {
	t.Parallel()

	// given
	udd := t.TempDir()
	store := filepath.Join(udd, "Default", "Cookies")
	writeFile(t, store, "")

	// when
	got := (chromiumFamily{}).version(store)

	// then
	if got != "" {
		t.Fatalf("major = %q, want empty", got)
	}
}

func TestFormatChromiumUA(t *testing.T) {
	t.Parallel()

	// given
	cases := []struct {
		name    string
		browser cookies.Browser
		goos    string
		major   string
		want    string
	}{
		{
			"chrome linux", cookies.BrowserChrome, "linux", "150",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"chrome macos", cookies.BrowserChrome, "darwin", "150",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"chrome windows", cookies.BrowserChrome, "windows", "150",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"edge appends Edg token", cookies.BrowserEdge, "windows", "150",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36 Edg/150.0.0.0",
		},
		{
			"brave has no fork token", cookies.BrowserBrave, "linux", "150",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"empty major falls back", cookies.BrowserChrome, "linux", "",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// when
			got := (chromiumFamily{browser: tc.browser}).userAgent(tc.goos, tc.major)

			// then
			if got != tc.want {
				t.Fatalf("\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
