package browserprovider

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type firefoxFamily struct{}

// version reads compatibility.ini's LastVersion; forks report their Gecko/ESR base.
func (firefoxFamily) version(storePath string) string {
	if storePath == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(storePath), "compatibility.ini"))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "LastVersion="); ok {
			version, _, _ := strings.Cut(value, "_")
			return majorOf(version)
		}
	}
	return ""
}

// userAgent builds a Firefox UA; forks send plain Firefox. https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/User-Agent/Firefox
func (firefoxFamily) userAgent(goos, major string) string {
	major = cmp.Or(major, "152") // fallback; bump to current stable periodically
	platform := "X11; Linux x86_64"
	switch goos {
	case "windows":
		platform = "Windows NT 10.0; Win64; x64"
	case "darwin":
		platform = "Macintosh; Intel Mac OS X 10.15"
	}
	return fmt.Sprintf("Mozilla/5.0 (%s; rv:%s.0) Gecko/20100101 Firefox/%s.0", platform, major, major)
}
