package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultConfigFile(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "  /tmp/xdg-config  ")
		t.Setenv("HOME", "/tmp/home")

		got := DefaultConfigFile()
		want := "/tmp/xdg-config/gh/attach.yml"
		if got != want {
			t.Fatalf("DefaultConfigFile() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to HOME when XDG_CONFIG_HOME is empty", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/tmp/home")

		got := DefaultConfigFile()
		want := "/tmp/home/.config/gh/attach.yml"
		if got != want {
			t.Fatalf("DefaultConfigFile() = %q, want %q", got, want)
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("returns empty config when file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "does-not-exist.yml")

		got, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v, want nil", err)
		}
		if !reflect.DeepEqual(got, Config{}) {
			t.Fatalf("LoadConfig() = %#v, want %#v", got, Config{})
		}
	})

	t.Run("parses attach.yml browsers format", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "attach.yml")
		content := `browsers:
  - browser: " chrome "
    profile: " Default "
    cookie_store_path: " /tmp/chrome-cookies "
  - browser: "firefox"
    profile: " default-release "
  - browser: " safari "
`
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		got, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v, want nil", err)
		}

		want := Config{
			Browsers: []BrowserEntry{
				{Browser: "chrome", Profile: "Default", CookieStorePath: "/tmp/chrome-cookies"},
				{Browser: "firefox", Profile: "default-release", CookieStorePath: ""},
				{Browser: "safari", Profile: "", CookieStorePath: ""},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("LoadConfig() = %#v, want %#v", got, want)
		}
	})

	t.Run("returns empty browsers when key is missing or empty", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			content string
		}{
			{name: "missing key", content: "foo: bar\n"},
			{name: "empty list", content: "browsers: []\n"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "attach.yml")
				if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
					t.Fatalf("write test file: %v", err)
				}

				got, err := LoadConfig(path)
				if err != nil {
					t.Fatalf("LoadConfig() error = %v, want nil", err)
				}
				if len(got.Browsers) != 0 {
					t.Fatalf("LoadConfig().Browsers len = %d, want 0", len(got.Browsers))
				}
			})
		}
	})

	t.Run("returns parse error on invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "attach.yml")
		if err := os.WriteFile(path, []byte("browsers: [\n"), 0o644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		_, err := LoadConfig(path)
		if err == nil {
			t.Fatalf("LoadConfig() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "parse config file") {
			t.Fatalf("LoadConfig() error = %q, want to contain %q", err.Error(), "parse attach file")
		}
	})
}
