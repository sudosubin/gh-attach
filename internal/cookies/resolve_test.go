package cookies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSources_CLIOverride(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "attach.yml")
	if err := os.WriteFile(path, []byte("browsers: [\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	sources, err := ResolveSources(ResolveInput{
		Browser:         "chrome",
		Profile:         "Default",
		CookieStorePath: "/tmp/Cookies",
		ConfigFile:      path,
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

func TestResolveSources_FromConfigFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "attach.yml")

	content := `browsers:
  - browser: chrome
    profile: Default
  - browser: firefox
    profile: default-release
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	sources, err := ResolveSources(ResolveInput{ConfigFile: path})
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
	t.Parallel()
	sources, err := ResolveSources(ResolveInput{ConfigFile: filepath.Join(t.TempDir(), "missing.yml")})
	if err != nil {
		t.Fatalf("ResolveSources() error = %v", err)
	}
	if len(sources) != 1 || sources[0].Browser != BrowserAuto {
		t.Fatalf("sources = %#v, want [auto]", sources)
	}
}

func TestResolveSources_InvalidBrowserInConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "attach.yml")
	content := `browsers:
  - browser: unknown-browser
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ResolveSources(ResolveInput{ConfigFile: path})
	if err == nil {
		t.Fatalf("ResolveSources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "config.browsers[0].browser") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "config.browsers[0].browser")
	}
}

func TestResolveSources_MissingBrowserInConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "attach.yml")
	content := `browsers:
  - profile: Default
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ResolveSources(ResolveInput{ConfigFile: path})
	if err == nil {
		t.Fatalf("ResolveSources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "config.browsers[0].browser: required") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "config.browsers[0].browser: required")
	}
}
