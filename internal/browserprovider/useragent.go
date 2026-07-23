package browserprovider

import (
	"strings"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

// browserFamily resolves a browser's version and assembles its User-Agent.
type browserFamily interface {
	version(storePath string) string
	userAgent(goos, version string) string
}

func familyFor(b cookies.Browser) browserFamily {
	switch {
	case b == cookies.BrowserSafari:
		return safariFamily{}
	case b.IsFirefox():
		return firefoxFamily{}
	case b == cookies.BrowserInline:
		return chromiumFamily{browser: cookies.BrowserChrome}
	default:
		return chromiumFamily{browser: b}
	}
}

// UserAgent assembles the browser's User-Agent, reading its version from storePath (Safari uses its app bundle).
func UserAgent(b cookies.Browser, goos, storePath string) string {
	f := familyFor(b)
	return f.userAgent(goos, f.version(storePath))
}

// majorOf returns the leading numeric component of a version, or "" if non-numeric.
func majorOf(version string) string {
	major, _, _ := strings.Cut(strings.TrimSpace(version), ".")
	if major == "" || strings.ContainsFunc(major, func(r rune) bool { return r < '0' || r > '9' }) {
		return ""
	}
	return major
}
