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

func TestResolveSources_CLIOverrideWithProfile(t *testing.T) {
	sources, err := ResolveSources(ResolveInput{
		Browser: "chrome",
		Profile: "Default",
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
	if sources[0].CookieStorePath != "" {
		t.Fatalf("cookieStorePath = %q, want empty", sources[0].CookieStorePath)
	}
}

func TestResolveSources_CLIOverrideWithCookieStorePath(t *testing.T) {
	sources, err := ResolveSources(ResolveInput{
		Browser:         "chrome",
		CookieStorePath: "/tmp/Cookies",
	})
	if err != nil {
		t.Fatalf("ResolveSources() error = %v", err)
	}

	if len(sources) != 1 || sources[0].CookieStorePath != "/tmp/Cookies" || sources[0].Profile != "" {
		t.Fatalf("sources = %#v", sources)
	}
}

func TestResolveSources_RejectsProfileWithCookieStorePathCLI(t *testing.T) {
	_, err := ResolveSources(ResolveInput{
		Browser:         "chrome",
		Profile:         "Default",
		CookieStorePath: "/tmp/Cookies",
	})
	if err == nil {
		t.Fatalf("ResolveSources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("error = %q, want to mention 'cannot be combined'", err.Error())
	}
}

func TestResolveSources_RejectsProfileWithCookieStorePathConfig(t *testing.T) {
	writeDefaultAttachConfig(t, `browsers:
  - browser: chrome
    profile: Default
    cookie_store_path: /tmp/Cookies
`)

	_, err := ResolveSources(ResolveInput{})
	if err == nil {
		t.Fatalf("ResolveSources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "config.browsers[0]") || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("error = %q, want to mention config.browsers[0] and 'cannot be combined'", err.Error())
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

func TestResolveSources_CookiesFileIsSoleSource(t *testing.T) {
	writeDefaultAttachConfig(t, `browsers:
  - browser: firefox
    profile: default-release
`)

	sources, err := ResolveSources(ResolveInput{
		CookiesFile: "  /tmp/github-cookies.json  ",
	})
	if err != nil {
		t.Fatalf("ResolveSources() error = %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("len(sources) = %d, want 1", len(sources))
	}
	if sources[0].Browser != BrowserInline {
		t.Fatalf("Browser = %q, want %q", sources[0].Browser, BrowserInline)
	}
	if sources[0].CookiesFile != "/tmp/github-cookies.json" {
		t.Fatalf("CookiesFile = %q", sources[0].CookiesFile)
	}
	if sources[0].Profile != "" || sources[0].CookieStorePath != "" {
		t.Fatalf("unexpected browser selectors: %#v", sources[0])
	}
}

func TestResolveSources_RejectsBlankCookiesFile(t *testing.T) {
	_, err := ResolveSources(ResolveInput{CookiesFile: "   "})
	if err == nil {
		t.Fatalf("ResolveSources() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--cookies-file cannot be empty") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestResolveSources_RejectsCookiesFileConflicts(t *testing.T) {
	cases := []struct {
		name string
		in   ResolveInput
		flag string
	}{
		{
			name: "browser",
			in: ResolveInput{
				CookiesFile: "/tmp/github-cookies.json",
				Browser:     "edge",
			},
			flag: "--browser",
		},
		{
			name: "profile",
			in: ResolveInput{
				CookiesFile: "/tmp/github-cookies.json",
				Profile:     "Profile 2",
			},
			flag: "--profile",
		},
		{
			name: "cookie store",
			in: ResolveInput{
				CookiesFile:     "/tmp/github-cookies.json",
				CookieStorePath: "/tmp/Cookies",
			},
			flag: "--cookie-store-path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveSources(tc.in)
			if err == nil {
				t.Fatalf("ResolveSources() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), "--cookies-file cannot be combined with") ||
				!strings.Contains(err.Error(), tc.flag) {
				t.Fatalf("error = %q, want conflict with %s", err.Error(), tc.flag)
			}
		})
	}
}
