package browserprovider

import (
	"os"
	"path/filepath"
	"strings"
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

func TestUserAgent_ReadsInstalledVersion(t *testing.T) {
	t.Parallel()

	// given
	udd := t.TempDir()
	store := filepath.Join(udd, "Default", "Cookies")
	writeFile(t, store, "")
	writeFile(t, filepath.Join(udd, "Last Version"), "151.0.1.2")

	// when
	got := UserAgent(cookies.BrowserChrome, "linux", store)

	// then
	if !strings.Contains(got, "Chrome/151.0.0.0") {
		t.Fatalf("UA did not use read version: %s", got)
	}
}
