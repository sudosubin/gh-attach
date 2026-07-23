package browserprovider

import (
	"context"
	"fmt"
	"runtime"

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

	// The version is install-wide, so one store path serves every session.
	userAgent := UserAgent(p.browser, runtime.GOOS, firstStorePath(sets))

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

func firstStorePath(sets []CookieSet) string {
	for _, s := range sets {
		if s.StorePath != "" {
			return s.StorePath
		}
	}
	return ""
}

func NewDefaultRegistry() map[cookies.Browser]BrowserProvider {
	sweet := newSweetcookieBackend()

	reg := make(map[cookies.Browser]BrowserProvider)
	for _, b := range cookies.ConcreteBrowsers() {
		reg[b] = &browserProvider{browser: b, backend: sweet}
	}
	reg[cookies.BrowserInline] = &browserProvider{
		browser: cookies.BrowserInline,
		backend: sweet,
	}
	return reg
}
