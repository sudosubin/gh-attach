package cookies

import (
	"fmt"
	"slices"
	"strings"
)

type Browser string

const (
	BrowserAuto Browser = "auto"

	BrowserArc       Browser = "arc"
	BrowserAtlas     Browser = "atlas"
	BrowserBrave     Browser = "brave"
	BrowserChrome    Browser = "chrome"
	BrowserChromium  Browser = "chromium"
	BrowserComet     Browser = "comet"
	BrowserDia       Browser = "dia"
	BrowserEdge      Browser = "edge"
	BrowserFirefox   Browser = "firefox"
	BrowserFloorp    Browser = "floorp"
	BrowserHelium    Browser = "helium"
	BrowserLibreWolf Browser = "librewolf"
	BrowserOpera     Browser = "opera"
	BrowserSafari    Browser = "safari"
	BrowserVivaldi   Browser = "vivaldi"
	BrowserWaterfox  Browser = "waterfox"
	BrowserWhale     Browser = "whale"
	BrowserZen       Browser = "zen"
)

// validBrowsers is derived from allBrowsers plus "auto".
var validBrowsers = func() map[Browser]struct{} {
	m := map[Browser]struct{}{BrowserAuto: {}}
	for _, b := range allBrowsers {
		m[b] = struct{}{}
	}
	return m
}()

type Source struct {
	Browser         Browser
	Profile         string
	CookieStorePath string
}

// allBrowsers is the canonical set of concrete (non-auto) browsers, alphabetical.
var allBrowsers = []Browser{
	BrowserArc,
	BrowserAtlas,
	BrowserBrave,
	BrowserChrome,
	BrowserChromium,
	BrowserComet,
	BrowserDia,
	BrowserEdge,
	BrowserFirefox,
	BrowserFloorp,
	BrowserHelium,
	BrowserLibreWolf,
	BrowserOpera,
	BrowserSafari,
	BrowserVivaldi,
	BrowserWaterfox,
	BrowserWhale,
	BrowserZen,
}

// autoProbeOrder is the probe order for "auto", ranked by macOS/Linux usage.
var autoProbeOrder = []Browser{
	// tier 1
	BrowserChrome,
	BrowserSafari,
	BrowserFirefox,
	// tier 2
	BrowserChromium,
	BrowserBrave,
	BrowserArc,
	BrowserVivaldi,
	BrowserEdge,
	BrowserOpera,
	// tier 3
	BrowserAtlas,
	BrowserComet,
	BrowserDia,
	BrowserHelium,
	BrowserWhale,
	BrowserFloorp,
	BrowserLibreWolf,
	BrowserWaterfox,
	BrowserZen,
}

func ExpandSource(source Source) []Source {
	if source.Browser != BrowserAuto {
		return []Source{source}
	}

	expanded := make([]Source, 0, len(autoProbeOrder))
	for _, b := range autoProbeOrder {
		expanded = append(expanded, Source{
			Browser:         b,
			Profile:         source.Profile,
			CookieStorePath: source.CookieStorePath,
		})
	}
	return expanded
}

func ApplyDefaultProfile(source Source) Source {
	if strings.TrimSpace(source.Profile) != "" || strings.TrimSpace(source.CookieStorePath) != "" {
		return source
	}

	if source.Browser.IsChromium() {
		source.Profile = "Default"
	}

	return source
}

func ParseBrowser(v string) (Browser, error) {
	b := Browser(strings.ToLower(strings.TrimSpace(v)))
	if b == "" {
		return BrowserAuto, nil
	}
	if _, ok := validBrowsers[b]; !ok {
		return "", fmt.Errorf("unsupported browser %q", v)
	}
	return b, nil
}

// IsChromium reports whether the browser uses the Chromium cookie store layout.
func (b Browser) IsChromium() bool {
	switch b {
	case BrowserArc, BrowserAtlas, BrowserBrave, BrowserChrome, BrowserChromium, BrowserComet,
		BrowserDia, BrowserEdge, BrowserHelium, BrowserOpera, BrowserVivaldi, BrowserWhale:
		return true
	}
	return false
}

// IsFirefox reports whether the browser is Firefox or a Firefox fork (with
// multi-account container support).
func (b Browser) IsFirefox() bool {
	switch b {
	case BrowserFirefox, BrowserFloorp, BrowserLibreWolf, BrowserWaterfox, BrowserZen:
		return true
	}
	return false
}

// ConcreteBrowsers returns every selectable browser except "auto".
func ConcreteBrowsers() []Browser {
	return slices.Clone(allBrowsers)
}

// BrowserChoices returns the pipe-separated browser names for help text.
func BrowserChoices() string {
	names := make([]string, 0, len(allBrowsers)+1)
	names = append(names, string(BrowserAuto))
	for _, b := range allBrowsers {
		names = append(names, string(b))
	}
	return strings.Join(names, "|")
}

// ProfileSelector is parsed from "<profile>[:<container-selector>]".
// Container selector formats:
//
//	""       — no container filter
//	"<name>" — match container name
//	"id=<N>" — match container id
type ProfileSelector struct {
	Profile   string
	Container string
	MatchByID bool
}

const profileContainerIDPrefix = "id="

// ParseProfileSelector splits on the first ':' only.
func ParseProfileSelector(s string) ProfileSelector {
	s = strings.TrimSpace(s)
	profile, container, hasContainer := strings.Cut(s, ":")
	if !hasContainer {
		return ProfileSelector{Profile: profile}
	}
	if v, ok := strings.CutPrefix(container, profileContainerIDPrefix); ok {
		return ProfileSelector{Profile: profile, Container: v, MatchByID: true}
	}
	return ProfileSelector{Profile: profile, Container: container}
}

// FormatProfileSelector builds the profile identifier.
//
//	(profile, "",     _)     -> "profile"
//	(profile, "name", false) -> "profile:name"
//	(profile, "N",    true)  -> "profile:id=N"
func FormatProfileSelector(profile, container string, byID bool) string {
	if container == "" {
		return profile
	}
	if byID {
		return profile + ":" + profileContainerIDPrefix + container
	}
	return profile + ":" + container
}
