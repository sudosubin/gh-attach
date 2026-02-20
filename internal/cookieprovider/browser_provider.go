package cookieprovider

import (
	"context"
	"fmt"
	"net/http"

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
		return ""
	}
	return p.backend.Name()
}

func (p *browserProvider) Load(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error) {
	if p.backend == nil {
		return nil, fmt.Errorf("cookie backend not configured for browser %s", p.browser)
	}

	source.Browser = p.browser
	return p.backend.Load(ctx, host, source)
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
