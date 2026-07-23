package browserprovider

import (
	"path/filepath"
	"testing"
)

func TestFirefoxMajorVersion(t *testing.T) {
	t.Parallel()

	// given
	cases := []struct {
		name        string
		lastVerLine string
		want        string
	}{
		{"firefox with build id", "LastVersion=152.0.6_20260713164047/20260713164047", "152"},
		{"waterfox reports esr base", "LastVersion=140.13.0_20260709161716/20260709161716", "140"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			dir := t.TempDir()
			store := filepath.Join(dir, "cookies.sqlite")
			writeFile(t, store, "")
			writeFile(t, filepath.Join(dir, "compatibility.ini"),
				"[Compatibility]\n"+tc.lastVerLine+"\nLastOSABI=Darwin_aarch64-gcc3\n")

			// when
			got := (firefoxFamily{}).version(store)

			// then
			if got != tc.want {
				t.Fatalf("major = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFirefoxMajorVersion_MissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()

	// given
	dir := t.TempDir()

	// when
	got := (firefoxFamily{}).version(filepath.Join(dir, "cookies.sqlite"))

	// then
	if got != "" {
		t.Fatalf("major = %q, want empty", got)
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
