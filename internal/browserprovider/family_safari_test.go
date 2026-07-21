package browserprovider

import (
	"path/filepath"
	"testing"
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
