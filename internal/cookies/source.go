package cookies

import (
	"fmt"
	"strings"
)

type Browser string

const (
	BrowserAuto     Browser = "auto"
	BrowserChrome   Browser = "chrome"
	BrowserChromium Browser = "chromium"
	BrowserEdge     Browser = "edge"
	BrowserFirefox  Browser = "firefox"
	BrowserSafari   Browser = "safari"
	BrowserBrave    Browser = "brave"
	BrowserVivaldi  Browser = "vivaldi"
	BrowserOpera    Browser = "opera"
)

var validBrowsers = map[Browser]struct{}{
	BrowserAuto:     {},
	BrowserChrome:   {},
	BrowserChromium: {},
	BrowserEdge:     {},
	BrowserFirefox:  {},
	BrowserSafari:   {},
	BrowserBrave:    {},
	BrowserVivaldi:  {},
	BrowserOpera:    {},
}

type Source struct {
	Browser         Browser
	Profile         string
	CookieStorePath string
}

var autoBrowserOrder = []Browser{
	BrowserChrome,
	BrowserChromium,
	BrowserEdge,
	BrowserBrave,
	BrowserVivaldi,
	BrowserOpera,
	BrowserFirefox,
	BrowserSafari,
}

func ExpandSource(source Source) []Source {
	if source.Browser != BrowserAuto {
		return []Source{source}
	}

	expanded := make([]Source, 0, len(autoBrowserOrder))
	for _, b := range autoBrowserOrder {
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

	switch source.Browser {
	case BrowserChrome,
		BrowserChromium,
		BrowserEdge,
		BrowserBrave,
		BrowserVivaldi,
		BrowserOpera:
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
