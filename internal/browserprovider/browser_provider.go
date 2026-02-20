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

func (p *browserProvider) Load(ctx context.Context, host string, source cookies.Source) (BrowserSession, error) {
	if p.backend == nil {
		return BrowserSession{}, fmt.Errorf("cookie backend not configured for browser %s", p.browser)
	}
	if p.browser == cookies.BrowserAuto || p.browser == "" {
		return BrowserSession{}, fmt.Errorf("browser provider requires concrete browser, got %q", p.browser)
	}

	source.Browser = p.browser
	loadedCookies, err := p.backend.Load(ctx, host, source)
	if err != nil {
		return BrowserSession{}, err
	}

	userAgent, err := UserAgentForBrowser(p.browser)
	if err != nil {
		return BrowserSession{}, err
	}

	return BrowserSession{
		Browser:   p.browser,
		Cookies:   loadedCookies,
		UserAgent: userAgent,
	}, nil
}

func NewDefaultRegistry() map[cookies.Browser]BrowserProvider {
	sweet := newSweetcookieBackend()
	kooky := newKookyBackend()

	return map[cookies.Browser]BrowserProvider{
		cookies.BrowserChrome:   &browserProvider{browser: cookies.BrowserChrome, backend: sweet},
		cookies.BrowserChromium: &browserProvider{browser: cookies.BrowserChromium, backend: sweet},
		cookies.BrowserEdge:     &browserProvider{browser: cookies.BrowserEdge, backend: sweet},
		cookies.BrowserBrave:    &browserProvider{browser: cookies.BrowserBrave, backend: sweet},
		cookies.BrowserVivaldi:  &browserProvider{browser: cookies.BrowserVivaldi, backend: sweet},
		cookies.BrowserOpera:    &browserProvider{browser: cookies.BrowserOpera, backend: sweet},
		cookies.BrowserFirefox:  &browserProvider{browser: cookies.BrowserFirefox, backend: kooky},
		cookies.BrowserSafari:   &browserProvider{browser: cookies.BrowserSafari, backend: kooky},
	}
}

func UserAgentForBrowser(browser cookies.Browser) (string, error) {
	gen := ua.WithSeed(0)

	switch browser {
	case cookies.BrowserChrome, cookies.BrowserChromium, cookies.BrowserBrave, cookies.BrowserVivaldi, cookies.BrowserOpera:
		switch runtime.GOOS {
		case "windows":
			return gen.ChromeWindows(), nil
		case "darwin":
			return gen.ChromeMac(), nil
		default:
			return gen.ChromeLinux(), nil
		}
	case cookies.BrowserEdge:
		if runtime.GOOS == "windows" {
			return gen.EdgeWindows(), nil
		} else {
			return gen.Edge(), nil
		}
	case cookies.BrowserFirefox:
		switch runtime.GOOS {
		case "windows":
			return gen.FirefoxWindows(), nil
		case "darwin":
			return gen.FirefoxMac(), nil
		default:
			return gen.Firefox(), nil
		}
	case cookies.BrowserSafari:
		return gen.Safari(), nil
	default:
		return gen.Chrome(), nil
	}
}
