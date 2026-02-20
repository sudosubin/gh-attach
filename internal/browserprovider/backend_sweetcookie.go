package browserprovider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	libsweetcookie "github.com/steipete/sweetcookie"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type sweetcookieBackend struct{}

func newSweetcookieBackend() *sweetcookieBackend {
	return &sweetcookieBackend{}
}

func (b *sweetcookieBackend) Name() string {
	return "sweetcookie"
}

func (b *sweetcookieBackend) Load(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error) {
	scBrowser, err := toSweetcookieBrowser(source.Browser)
	if err != nil {
		return nil, err
	}

	opts := libsweetcookie.Options{
		URL:      "https://" + host + "/",
		Browsers: []libsweetcookie.Browser{scBrowser},
		Mode:     libsweetcookie.ModeMerge,
		Timeout:  30 * time.Second,
	}

	override := source.Profile
	if source.CookieStorePath != "" {
		override = source.CookieStorePath
	}
	if override != "" {
		opts.Profiles = map[libsweetcookie.Browser]string{scBrowser: override}
	}

	result, err := libsweetcookie.Get(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(result.Cookies) == 0 {
		return nil, fmt.Errorf("no cookies found")
	}

	out := make([]*http.Cookie, 0, len(result.Cookies))
	for _, c := range result.Cookies {
		hc := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}
		if c.Expires != nil {
			hc.Expires = *c.Expires
		}
		out = append(out, hc)
	}

	return out, nil
}

func toSweetcookieBrowser(browser cookies.Browser) (libsweetcookie.Browser, error) {
	switch browser {
	case cookies.BrowserChrome:
		return libsweetcookie.BrowserChrome, nil
	case cookies.BrowserChromium:
		return libsweetcookie.BrowserChromium, nil
	case cookies.BrowserEdge:
		return libsweetcookie.BrowserEdge, nil
	case cookies.BrowserBrave:
		return libsweetcookie.BrowserBrave, nil
	case cookies.BrowserVivaldi:
		return libsweetcookie.BrowserVivaldi, nil
	case cookies.BrowserOpera:
		return libsweetcookie.BrowserOpera, nil
	default:
		return "", fmt.Errorf("unsupported sweetcookie browser %s", browser)
	}
}
