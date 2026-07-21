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
