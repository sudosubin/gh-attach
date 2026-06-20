package cookies

import "testing"

func TestExpandSource_Auto(t *testing.T) {
	t.Parallel()
	s := Source{Browser: BrowserAuto, Profile: "Default"}
	expanded := ExpandSource(s)
	if len(expanded) == 0 {
		t.Fatalf("expected expanded sources")
	}
	if expanded[0].Browser != BrowserArc {
		t.Fatalf("first browser = %s", expanded[0].Browser)
	}
	if expanded[len(expanded)-1].Browser != BrowserZen {
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
		BrowserArc,
		BrowserHelium,
		BrowserDia,
		BrowserComet,
		BrowserAtlas,
		BrowserWhale,
	} {
		got := ApplyDefaultProfile(Source{Browser: b})
		if got.Profile != "Default" {
			t.Fatalf("browser=%s profile=%q, want %q", b, got.Profile, "Default")
		}
	}
}

func TestApplyDefaultProfile_FirefoxFamilyHasNoDefault(t *testing.T) {
	t.Parallel()

	for _, b := range []Browser{
		BrowserFirefox,
		BrowserZen,
		BrowserFloorp,
		BrowserWaterfox,
		BrowserLibreWolf,
	} {
		got := ApplyDefaultProfile(Source{Browser: b})
		if got.Profile != "" {
			t.Fatalf("browser=%s profile=%q, want empty", b, got.Profile)
		}
	}
}

func TestBrowserFamilies(t *testing.T) {
	t.Parallel()

	for _, b := range ConcreteBrowsers() {
		chromium := b.IsChromium()
		firefox := b.IsFirefox()
		if chromium && firefox {
			t.Fatalf("browser=%s classified as both chromium and firefox", b)
		}
		// safari is neither; every other concrete browser is exactly one family.
		if b != BrowserSafari && !chromium && !firefox {
			t.Fatalf("browser=%s classified as no family", b)
		}
	}
	if BrowserSafari.IsChromium() || BrowserSafari.IsFirefox() {
		t.Fatalf("safari should not belong to a family")
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

func TestParseProfileSelector(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want ProfileSelector
	}{
		{"empty", "", ProfileSelector{}},
		{"profile only", "default", ProfileSelector{Profile: "default"}},
		{"profile + name", "default:Work", ProfileSelector{Profile: "default", Container: "Work"}},
		{"profile + id", "default:id=2", ProfileSelector{Profile: "default", Container: "2", MatchByID: true}},
		{"profile + name with email-like value", "default:sudosubin@example.com", ProfileSelector{Profile: "default", Container: "sudosubin@example.com"}},
		{"empty profile + name", ":Work", ProfileSelector{Profile: "", Container: "Work"}},
		{"profile with empty container suffix", "default:", ProfileSelector{Profile: "default"}},
		{"profile + name containing colon", "default:scope:value", ProfileSelector{Profile: "default", Container: "scope:value"}},
		{"trims surrounding whitespace", "  default:Work  ", ProfileSelector{Profile: "default", Container: "Work"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseProfileSelector(tc.in)
			if got != tc.want {
				t.Fatalf("ParseProfileSelector(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormatProfileSelector(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		profile   string
		container string
		byID      bool
		want      string
	}{
		{"profile only", "default", "", false, "default"},
		{"profile only with byID ignored", "default", "", true, "default"},
		{"profile + name", "default", "Work", false, "default:Work"},
		{"profile + id", "default", "2", true, "default:id=2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatProfileSelector(tc.profile, tc.container, tc.byID)
			if got != tc.want {
				t.Fatalf("FormatProfileSelector(%q,%q,%v) = %q, want %q", tc.profile, tc.container, tc.byID, got, tc.want)
			}
		})
	}
}

func TestProfileSelectorRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []string{
		"default",
		"default:Work",
		"default:id=2",
		"default:sudosubin@example.com",
	}

	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			sel := ParseProfileSelector(in)
			got := FormatProfileSelector(sel.Profile, sel.Container, sel.MatchByID)
			if got != in {
				t.Fatalf("round-trip %q -> %q", in, got)
			}
		})
	}
}
