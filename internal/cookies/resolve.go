package cookies

import (
	"fmt"
	"strings"

	"github.com/sudosubin/gh-attach/internal/config"
)

type ResolveInput struct {
	Browser         string
	Profile         string
	CookieStorePath string
}

func ResolveSources(in ResolveInput) ([]Source, error) {
	if hasCLIOverride(in) {
		browser, err := ParseBrowser(in.Browser)
		if err != nil {
			return nil, err
		}
		return []Source{{
			Browser:         browser,
			Profile:         strings.TrimSpace(in.Profile),
			CookieStorePath: strings.TrimSpace(in.CookieStorePath),
		}}, nil
	}

	path := config.DefaultConfigFile()

	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	if len(cfg.Browsers) == 0 {
		return []Source{{Browser: BrowserAuto}}, nil
	}

	sources := make([]Source, 0, len(cfg.Browsers))
	for i, e := range cfg.Browsers {
		if strings.TrimSpace(e.Browser) == "" {
			return nil, fmt.Errorf("config.browsers[%d].browser: required", i)
		}

		browser, err := ParseBrowser(e.Browser)
		if err != nil {
			return nil, fmt.Errorf("config.browsers[%d].browser: %w", i, err)
		}
		sources = append(sources, Source{
			Browser:         browser,
			Profile:         e.Profile,
			CookieStorePath: e.CookieStorePath,
		})
	}

	return sources, nil
}

func hasCLIOverride(in ResolveInput) bool {
	return strings.TrimSpace(in.Browser) != "" ||
		strings.TrimSpace(in.Profile) != "" ||
		strings.TrimSpace(in.CookieStorePath) != ""
}
