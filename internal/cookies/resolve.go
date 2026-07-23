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
	CookiesFile     string
}

func ResolveSources(in ResolveInput) ([]Source, error) {
	cookiesFile := strings.TrimSpace(in.CookiesFile)
	if in.CookiesFile != "" {
		if cookiesFile == "" {
			return nil, fmt.Errorf("--cookies-file cannot be empty")
		}

		conflicts := make([]string, 0, 3)
		if strings.TrimSpace(in.Browser) != "" {
			conflicts = append(conflicts, "--browser")
		}
		if strings.TrimSpace(in.Profile) != "" {
			conflicts = append(conflicts, "--profile")
		}
		if strings.TrimSpace(in.CookieStorePath) != "" {
			conflicts = append(conflicts, "--cookie-store-path")
		}
		if len(conflicts) > 0 {
			return nil, fmt.Errorf(
				"--cookies-file cannot be combined with %s",
				strings.Join(conflicts, ", "),
			)
		}

		return []Source{{
			Browser:     BrowserInline,
			CookiesFile: cookiesFile,
		}}, nil
	}

	if hasCLIOverride(in) {
		profile := strings.TrimSpace(in.Profile)
		path := strings.TrimSpace(in.CookieStorePath)
		if profile != "" && path != "" {
			return nil, fmt.Errorf("--profile cannot be combined with --cookie-store-path; the path bypasses profile discovery")
		}
		browser, err := ParseBrowser(in.Browser)
		if err != nil {
			return nil, err
		}
		return []Source{{
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
		profile := strings.TrimSpace(e.Profile)
		storePath := strings.TrimSpace(e.CookieStorePath)
		if profile != "" && storePath != "" {
			return nil, fmt.Errorf("config.browsers[%d]: profile cannot be combined with cookie_store_path", i)
		}
		sources = append(sources, Source{
			Browser:         browser,
			Profile:         profile,
			CookieStorePath: storePath,
		})
	}

	return sources, nil
}

func hasCLIOverride(in ResolveInput) bool {
	return strings.TrimSpace(in.Browser) != "" ||
		strings.TrimSpace(in.Profile) != "" ||
		strings.TrimSpace(in.CookieStorePath) != ""
}
