package app

import (
	"fmt"
	"strings"

	"github.com/sudosubin/gh-attach/internal/config"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type cookieSourceInput struct {
	Browser         string
	Profile         string
	CookieStorePath string
}

func resolveCookieSources(in cookieSourceInput) ([]cookies.Source, error) {
	if hasCookieSourceOverride(in) {
		profile := strings.TrimSpace(in.Profile)
		path := strings.TrimSpace(in.CookieStorePath)
		if profile != "" && path != "" {
			return nil, fmt.Errorf("--profile cannot be combined with --cookie-store-path; the path bypasses profile discovery")
		}
		browser, err := cookies.ParseBrowser(in.Browser)
		if err != nil {
			return nil, err
		}
		return []cookies.Source{{
			Browser:         browser,
			Profile:         profile,
			CookieStorePath: path,
		}}, nil
	}

	path := config.DefaultConfigFile()

	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	if len(cfg.Browsers) == 0 {
		return []cookies.Source{{Browser: cookies.BrowserAuto}}, nil
	}

	sources := make([]cookies.Source, 0, len(cfg.Browsers))
	for i, e := range cfg.Browsers {
		if strings.TrimSpace(e.Browser) == "" {
			return nil, fmt.Errorf("config.browsers[%d].browser: required", i)
		}

		browser, err := cookies.ParseBrowser(e.Browser)
		if err != nil {
			return nil, fmt.Errorf("config.browsers[%d].browser: %w", i, err)
		}
		profile := strings.TrimSpace(e.Profile)
		storePath := strings.TrimSpace(e.CookieStorePath)
		if profile != "" && storePath != "" {
			return nil, fmt.Errorf("config.browsers[%d]: profile cannot be combined with cookie_store_path", i)
		}
		sources = append(sources, cookies.Source{
			Browser:         browser,
			Profile:         profile,
			CookieStorePath: storePath,
		})
	}

	return sources, nil
}

func hasCookieSourceOverride(in cookieSourceInput) bool {
	return strings.TrimSpace(in.Browser) != "" ||
		strings.TrimSpace(in.Profile) != "" ||
		strings.TrimSpace(in.CookieStorePath) != ""
}
