package cookies

import "testing"

func TestExpandSource_Auto(t *testing.T) {
	t.Parallel()
	s := Source{Browser: BrowserAuto, Profile: "Default"}
	expanded := ExpandSource(s)
	if len(expanded) == 0 {
		t.Fatalf("expected expanded sources")
	}
	if expanded[0].Browser != BrowserChrome {
		t.Fatalf("first browser = %s", expanded[0].Browser)
	}
	if expanded[len(expanded)-1].Browser != BrowserSafari {
		t.Fatalf("last browser = %s", expanded[len(expanded)-1].Browser)
	}
	for _, src := range expanded {
		if src.Profile != "Default" {
			t.Fatalf("profile not propagated: %#v", src)
		}
	}
}

func TestExpandSource_NonAuto(t *testing.T) {
	t.Parallel()
	s := Source{Browser: BrowserFirefox}
	expanded := ExpandSource(s)
	if len(expanded) != 1 {
		t.Fatalf("len(expanded) = %d", len(expanded))
	}
	if expanded[0].Browser != BrowserFirefox {
		t.Fatalf("browser = %s", expanded[0].Browser)
	}
}
