package browserprovider

import (
	"context"
	"fmt"
	"runtime"

	ua "github.com/nzrsky/useragent-generator/pkg/useragent"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type browserProvider struct {
	browser cookies.Browser
	backend Backend
}

func (p *browserProvider) Browser() cookies.Browser {
	return p.browser
}

func (p *browserProvider) BackendName() string {
	if p.backend == nil {
		return "unconfigured"
	}
	return p.backend.Name()
}

func (p *browserProvider) Load(ctx context.Context, host string, source cookies.Source) ([]BrowserSession, error) {
	if p.backend == nil {
		return nil, fmt.Errorf("cookie backend not configured for browser %s", p.browser)
	}
	if p.browser == cookies.BrowserAuto || p.browser == "" {
		return nil, fmt.Errorf("browser provider requires concrete browser, got %q", p.browser)
	}

	source.Browser = p.browser
	sets, err := p.backend.Load(ctx, host, source)
	if err != nil {
		return nil, err
	}

	userAgent, err := UserAgentForBrowser(p.browser)
	if err != nil {
		return nil, err
	}

	sessions := make([]BrowserSession, 0, len(sets))
	for _, s := range sets {
		sessions = append(sessions, BrowserSession{
			Browser:   p.browser,
			Profile:   s.Profile,
			Cookies:   s.Cookies,
			UserAgent: userAgent,
		})
	}
	return sessions, nil
}

func NewDefaultRegistry() map[cookies.Browser]BrowserProvider {
	sweet := newSweetcookieBackend()

	reg := make(map[cookies.Browser]BrowserProvider)
	for _, b := range cookies.ConcreteBrowsers() {
		reg[b] = &browserProvider{browser: b, backend: sweet}
	}
	return reg
}

func UserAgentForBrowser(browser cookies.Browser) (string, error) {
	gen := ua.WithSeed(0)

	switch {
	case browser == cookies.BrowserEdge:
		if runtime.GOOS == "windows" {
			return gen.EdgeWindows(), nil
		}
		return gen.Edge(), nil
	case browser.IsFirefox():
		switch runtime.GOOS {
		case "windows":
			return gen.FirefoxWindows(), nil
		case "darwin":
			return gen.FirefoxMac(), nil
		default:
			return gen.Firefox(), nil
		}
	case browser == cookies.BrowserSafari:
		return gen.Safari(), nil
	case browser.IsChromium():
		switch runtime.GOOS {
		case "windows":
			return gen.ChromeWindows(), nil
		case "darwin":
			return gen.ChromeMac(), nil
		default:
			return gen.ChromeLinux(), nil
		}
	default:
		return gen.Chrome(), nil
	}
}
