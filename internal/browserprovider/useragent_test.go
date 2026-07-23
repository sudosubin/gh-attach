package browserprovider

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestUserAgent_ReadsInstalledVersion(t *testing.T) {
	t.Parallel()

	// given
	udd := t.TempDir()
	store := filepath.Join(udd, "Default", "Cookies")
	writeFile(t, store, "")
	writeFile(t, filepath.Join(udd, "Last Version"), "151.0.1.2")

	// when
	got := UserAgent(cookies.BrowserChrome, "linux", store)

	// then
	if !strings.Contains(got, "Chrome/151.0.0.0") {
		t.Fatalf("UA did not use read version: %s", got)
	}
}

func TestUserAgent_DispatchesByFamily(t *testing.T) {
	t.Parallel()

	// given
	cases := []struct {
		browser  cookies.Browser
		contains string
		absent   string
	}{
		{cookies.BrowserBrave, "Chrome/150.0.0.0", "Edg/"},
		{cookies.BrowserArc, "Chrome/150.0.0.0", "Edg/"},
		{cookies.BrowserEdge, "Edg/150.0.0.0", ""},
		{cookies.BrowserLibreWolf, "Firefox/152.0", "Chrome/"},
		{cookies.BrowserWaterfox, "Firefox/152.0", "Chrome/"},
		{cookies.BrowserSafari, "Safari/605.1.15", "Chrome/"},
	}
	for _, tc := range cases {
		t.Run(string(tc.browser), func(t *testing.T) {
			t.Parallel()

			// when
			got := UserAgent(tc.browser, "linux", "")

			// then
			if !strings.Contains(got, tc.contains) {
				t.Fatalf("%s: %q missing from %s", tc.browser, tc.contains, got)
			}
			if tc.absent != "" && strings.Contains(got, tc.absent) {
				t.Fatalf("%s: %q should be absent from %s", tc.browser, tc.absent, got)
			}
		})
	}
}

func TestUserAgent_EveryBrowserNonEmpty(t *testing.T) {
	t.Parallel()

	// given
	for _, b := range cookies.ConcreteBrowsers() {
		for _, goos := range []string{"linux", "darwin", "windows"} {
			// when
			got := UserAgent(b, goos, "")

			// then
			if !strings.HasPrefix(got, "Mozilla/5.0 ") {
				t.Fatalf("browser=%s goos=%s produced invalid UA: %q", b, goos, got)
			}
		}
	}
}

func TestMajorOf(t *testing.T) {
	t.Parallel()

	// given
	cases := map[string]string{
		"150.0.7871.128": "150",
		"140.13.0":       "140",
		"152":            "152",
		"":               "",
		"  151.0  ":      "151",
		"v150":           "",
		"abc":            "",
	}
	for in, want := range cases {
		// when
		got := majorOf(in)

		// then
		if got != want {
			t.Fatalf("majorOf(%q) = %q, want %q", in, got, want)
		}
	}
}
