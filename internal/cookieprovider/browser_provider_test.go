package cookieprovider

import (
	"testing"

	"github.com/sudosubin/gh-attach/internal/cookies"
)

func TestNewDefaultRegistry(t *testing.T) {
	t.Parallel()

	reg := NewDefaultRegistry()

	if got := reg[cookies.BrowserChrome].BackendName(); got != "sweetcookie" {
		t.Fatalf("chrome backend = %q", got)
	}
	if got := reg[cookies.BrowserChromium].BackendName(); got != "sweetcookie" {
		t.Fatalf("chromium backend = %q", got)
	}
	if got := reg[cookies.BrowserFirefox].BackendName(); got != "kooky" {
		t.Fatalf("firefox backend = %q", got)
	}
	if got := reg[cookies.BrowserSafari].BackendName(); got != "kooky" {
		t.Fatalf("safari backend = %q", got)
	}
}
