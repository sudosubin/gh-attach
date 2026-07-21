package browserprovider

import (
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestFormatChromiumUA(t *testing.T) {
	t.Parallel()

	// given
	cases := []struct {
		name    string
		browser cookies.Browser
		goos    string
		major   string
		want    string
	}{
		{
			"chrome linux", cookies.BrowserChrome, "linux", "150",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"chrome macos", cookies.BrowserChrome, "darwin", "150",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"chrome windows", cookies.BrowserChrome, "windows", "150",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"edge appends Edg token", cookies.BrowserEdge, "windows", "150",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36 Edg/150.0.0.0",
		},
		{
			"brave has no fork token", cookies.BrowserBrave, "linux", "150",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
		{
			"empty major falls back", cookies.BrowserChrome, "linux", "",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// when
			got := (chromiumFamily{browser: tc.browser}).userAgent(tc.goos, tc.major)

			// then
			if got != tc.want {
				t.Fatalf("\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestFormatFirefoxUA(t *testing.T) {
	t.Parallel()

	// given
	cases := []struct {
		name        string
		goos, major string
		want        string
	}{
		{
			"linux", "linux", "152",
			"Mozilla/5.0 (X11; Linux x86_64; rv:152.0) Gecko/20100101 Firefox/152.0",
		},
		{
			"macos uses capped 10.15", "darwin", "152",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:152.0) Gecko/20100101 Firefox/152.0",
		},
		{
			"empty major falls back", "windows", "",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:152.0) Gecko/20100101 Firefox/152.0",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// when
			got := (firefoxFamily{}).userAgent(tc.goos, tc.major)

			// then
			if got != tc.want {
				t.Fatalf("\n got: %s\nwant: %s", got, tc.want)
			}
		})
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
