package browserprovider

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestSafariVersion(t *testing.T) {
	t.Parallel()

	// given
	plist := filepath.Join(t.TempDir(), "Info.plist")
	writeFile(t, plist, `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>Safari</string>
	<key>CFBundleShortVersionString</key>
	<string>26.5.2</string>
	<key>CFBundleVersion</key>
	<string>21624.2.5.11.8</string>
</dict>
</plist>`)

	// when
	got := safariVersion(plist)

	// then
	if got != "26.5.2" {
		t.Fatalf("safari version = %q, want 26.5.2", got)
	}

	// given
	// when
	missing := safariVersion(filepath.Join(t.TempDir(), "missing.plist"))
	// then
	if missing != "" {
		t.Fatalf("missing plist should yield empty, got %q", missing)
	}
}

func TestSafariVersion_RealBundle(t *testing.T) {
	t.Parallel()

	// given: the real macOS Safari app bundle
	const plist = "/Applications/Safari.app/Contents/Info.plist"
	if runtime.GOOS != "darwin" {
		t.Skip("Safari bundle exists only on macOS")
	}
	if _, err := os.Stat(plist); err != nil {
		t.Skip("Safari not installed")
	}

	// when: safariVersion parses the installed Info.plist
	got := safariVersion(plist)

	// then: it yields a dotted numeric version that flows into the User-Agent
	if !regexp.MustCompile(`^\d+(\.\d+)+$`).MatchString(got) {
		t.Fatalf("safari version = %q, want a dotted numeric version", got)
	}
	if ua := UserAgent(cookies.BrowserSafari, "darwin", ""); !strings.Contains(ua, "Version/"+got+" ") {
		t.Fatalf("UA %q missing Version/%s", ua, got)
	}
}

func TestFormatSafariUA(t *testing.T) {
	t.Parallel()

	// given
	// when
	got := (safariFamily{}).userAgent("", "26.5.2")
	// then
	if got != "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.5.2 Safari/605.1.15" {
		t.Fatalf("full version: %s", got)
	}

	// given
	// when
	fallback := (safariFamily{}).userAgent("", "")
	// then
	if fallback != "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15" {
		t.Fatalf("fallback: %s", fallback)
	}
}
