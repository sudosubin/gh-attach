package browserprovider

import (
	"cmp"
	"fmt"
	"os"
	"strings"
)

type safariFamily struct{}

// version reads Safari's version from its app bundle; storePath is unused.
func (safariFamily) version(string) string {
	return safariVersion("/Applications/Safari.app/Contents/Info.plist")
}

// userAgent builds a Safari UA; Safari applies no reduction, so the full version shows.
func (safariFamily) userAgent(_, version string) string {
	return fmt.Sprintf("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/%s Safari/605.1.15",
		cmp.Or(version, "26.0")) // fallback; bump to current stable periodically
}

// safariVersion reads CFBundleShortVersionString from Safari's Info.plist (absent off macOS); its <string> value follows the key.
func safariVersion(plistPath string) string {
	b, err := os.ReadFile(plistPath)
	if err != nil {
		return ""
	}
	_, after, ok := strings.Cut(string(b), "<key>CFBundleShortVersionString</key>")
	if !ok {
		return ""
	}
	_, after, ok = strings.Cut(after, "<string>")
	if !ok {
		return ""
	}
	version, _, ok := strings.Cut(after, "</string>")
	if !ok {
		return ""
	}
	return strings.TrimSpace(version)
}
