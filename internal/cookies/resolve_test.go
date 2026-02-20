package cookies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDefaultAttachConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", filepath.Join(dir, "home"))

	path := filepath.Join(dir, "gh", "attach.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	return path
}

func TestResolveSources_CLIOverride(t *testing.T) {
	sources, err := ResolveSources(ResolveInput{
		Browser:         "chrome",
		Profile:         "Default",
		CookieStorePath: "/tmp/Cookies",
	})
	if err != nil {
		t.Fatalf("ResolveSources() error = %v", err)
	}

	if len(sources) != 1 {
		t.Fatalf("len(sources) = %d, want 1", len(sources))
	}
	if sources[0].Browser != BrowserChrome {
		t.Fatalf("browser = %s, want %s", sources[0].Browser, BrowserChrome)
	}
	if sources[0].Profile != "Default" {
		t.Fatalf("profile = %q", sources[0].Profile)
	}
	if sources[0].CookieStorePath != "/tmp/Cookies" {
		t.Fatalf("cookieStorePath = %q", sources[0].CookieStorePath)
	}
}

func TestResolveSources_FromDefaultConfigFile(t *testing.T) {
	writeDefaultAttachConfig(t, `browsers:
  - browser: chrome
    profile: Default
  - browser: firefox
    profile: default-release
`)

	sources, err := ResolveSources(ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveSources() error = %v", err)
	}

	if len(sources) != 2 {
		t.Fatalf("len(sources) = %d, want 2", len(sources))
	}
	if sources[0].Browser != BrowserChrome || sources[1].Browser != BrowserFirefox {
		t.Fatalf("unexpected browser chain: %#v", sources)
	}
}

func TestResolveSources_DefaultAutoWhenNoConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", filepath.Join(dir, "home"))

	sources, err := ResolveSources(ResolveInput{})
	if err != nil {
		t.Fatalf("ResolveSources() error = %v", err)
	}
	if len(sources) != 1 || sources[0].Browser != BrowserAuto {
		t.Fatalf("sources = %#v, want [auto]", sources)
	}
}

func TestResolveSources_InvalidBrowserInConfig(t *testing.T) {
	writeDefaultAttachConfig(t, `browsers:
  - browser: unknown-browser
`)

	_, err := ResolveSources(ResolveInput{})
	if err == nil {
		t.Fatalf("ResolveSources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "config.browsers[0].browser") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "config.browsers[0].browser")
	}
}

func TestResolveSources_MissingBrowserInConfig(t *testing.T) {
	writeDefaultAttachConfig(t, `browsers:
  - profile: Default
`)

	_, err := ResolveSources(ResolveInput{})
	if err == nil {
		t.Fatalf("ResolveSources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "config.browsers[0].browser: required") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "config.browsers[0].browser: required")
	}
}
