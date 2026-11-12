package browserprovider

import (
	"cmp"
	"fmt"
	"os"
	"strings"

	"howett.net/plist"
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

// safariVersion reads CFBundleShortVersionString from an XML or binary Info.plist.
func safariVersion(plistPath string) string {
	b, err := os.ReadFile(plistPath)
	if err != nil {
		return ""
	}
	var info struct {
		Version string `plist:"CFBundleShortVersionString"`
	}
	if _, err := plist.Unmarshal(b, &info); err != nil {
		return ""
	}
	return strings.TrimSpace(info.Version)
}
