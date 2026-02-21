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

func TestApplyDefaultProfile_ChromiumFamily(t *testing.T) {
	t.Parallel()

	for _, b := range []Browser{
		BrowserChrome,
		BrowserChromium,
		BrowserEdge,
		BrowserBrave,
		BrowserVivaldi,
		BrowserOpera,
	} {
		got := ApplyDefaultProfile(Source{Browser: b})
		if got.Profile != "Default" {
			t.Fatalf("browser=%s profile=%q, want %q", b, got.Profile, "Default")
		}
	}
}

func TestApplyDefaultProfile_DoesNotOverrideExplicitProfileOrPath(t *testing.T) {
	t.Parallel()

	withProfile := ApplyDefaultProfile(Source{Browser: BrowserChromium, Profile: "Work"})
	if withProfile.Profile != "Work" {
		t.Fatalf("profile=%q, want %q", withProfile.Profile, "Work")
	}

	withPath := ApplyDefaultProfile(Source{Browser: BrowserChromium, CookieStorePath: "/tmp/Cookies"})
	if withPath.Profile != "" {
		t.Fatalf("profile=%q, want empty", withPath.Profile)
	}

	firefox := ApplyDefaultProfile(Source{Browser: BrowserFirefox})
	if firefox.Profile != "" {
		t.Fatalf("firefox profile=%q, want empty", firefox.Profile)
	}
}
