package browserprovider

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	libsweetcookie "github.com/steipete/sweetcookie"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type sweetcookieBackend struct{}

type inlineSafeError struct {
	message string
}

func (e inlineSafeError) Error() string {
	return e.message
}

func (e inlineSafeError) SafeMessage() string {
	return e.message
}

func newSweetcookieBackend() *sweetcookieBackend {
	return &sweetcookieBackend{}
}

func (*sweetcookieBackend) Name() string {
	return "sweetcookie"
}

func (b *sweetcookieBackend) Load(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
	if source.Browser == cookies.BrowserInline {
		return b.loadInline(ctx, host, source)
	}
	if source.Browser.IsFirefox() {
		return b.loadFirefox(ctx, host, source)
	}
	return b.loadMerged(ctx, host, source)
}

func (*sweetcookieBackend) loadInline(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
	if err := validateInlineCookieFile(source.CookiesFile); err != nil {
		return nil, err
	}

	result, err := libsweetcookie.Get(ctx, libsweetcookie.Options{
		URL:      "https://" + host + "/",
		Browsers: []libsweetcookie.Browser{libsweetcookie.BrowserInline},
		Mode:     libsweetcookie.ModeFirst,
		Inline: libsweetcookie.InlineCookies{
			File: source.CookiesFile,
		},
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Cookies) == 0 {
		return nil, inlineSafeError{message: fmt.Sprintf("no usable %s cookies found in --cookies-file", host)}
	}

	out := make([]*http.Cookie, 0, len(result.Cookies))
	for _, cookie := range result.Cookies {
		out = append(out, httpCookie(cookie))
	}
	return []CookieSet{{Cookies: out}}, nil
}

func validateInlineCookieFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return inlineSafeError{message: "read --cookies-file: file does not exist"}
		}
		if os.IsPermission(err) {
			return inlineSafeError{message: "read --cookies-file: permission denied"}
		}
		return inlineSafeError{message: "read --cookies-file: unable to read"}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return inlineSafeError{message: "--cookies-file is empty"}
	}
	if !json.Valid(raw) {
		return inlineSafeError{message: "--cookies-file contains invalid JSON"}
	}
	return nil
}

// loadMerged reads a single profile's cookies as one set (Chromium family, Safari).
func (*sweetcookieBackend) loadMerged(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
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
	if override := cmp.Or(source.CookieStorePath, source.Profile); override != "" {
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
		out = append(out, httpCookie(c))
	}
	return []CookieSet{{
		Profile:   source.Profile,
		Cookies:   out,
		StorePath: result.Cookies[0].Source.StorePath,
	}}, nil
}

// loadFirefox reads Firefox cookies and groups them by profile and multi-account
// container, applying the "<profile>[:<container>]" selector.
func (*sweetcookieBackend) loadFirefox(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
	scBrowser, err := toSweetcookieBrowser(source.Browser)
	if err != nil {
		return nil, err
	}
	sel := cookies.ParseProfileSelector(source.Profile)

	opts := libsweetcookie.Options{
		URL:      "https://" + host + "/",
		Browsers: []libsweetcookie.Browser{scBrowser},
		Mode:     libsweetcookie.ModeMerge,
		Timeout:  30 * time.Second,
	}
	// Let sweetcookie pin the store (explicit path) or narrow to the profile;
	// only container filtering is applied below.
	if override := cmp.Or(source.CookieStorePath, sel.Profile); override != "" {
		opts.Profiles = map[libsweetcookie.Browser]string{scBrowser: override}
	}

	result, err := libsweetcookie.Get(ctx, opts)
	if err != nil {
		return nil, err
	}

	groups := map[containerGroupKey][]*http.Cookie{}
	for _, c := range result.Cookies {
		key, ok := cookieGroupKey(sel, c.Source.Profile, c.Container)
		if !ok {
			continue
		}
		groups[key] = append(groups[key], httpCookie(c))
	}

	sets := finalizeContainerGroups(groups)
	if len(sets) == 0 {
		return nil, fmt.Errorf("no cookies found")
	}
	// One store path serves all sets: they share the same Firefox install.
	storePath := result.Cookies[0].Source.StorePath
	for i := range sets {
		sets[i].StorePath = storePath
	}
	return sets, nil
}

func httpCookie(c libsweetcookie.Cookie) *http.Cookie {
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
	return hc
}

func toSweetcookieBrowser(browser cookies.Browser) (libsweetcookie.Browser, error) {
	switch browser {
	case cookies.BrowserArc:
		return libsweetcookie.BrowserArc, nil
	case cookies.BrowserAtlas:
		return libsweetcookie.BrowserAtlas, nil
	case cookies.BrowserBrave:
		return libsweetcookie.BrowserBrave, nil
	case cookies.BrowserChrome:
		return libsweetcookie.BrowserChrome, nil
	case cookies.BrowserChromium:
		return libsweetcookie.BrowserChromium, nil
	case cookies.BrowserComet:
		return libsweetcookie.BrowserComet, nil
	case cookies.BrowserDia:
		return libsweetcookie.BrowserDia, nil
	case cookies.BrowserEdge:
		return libsweetcookie.BrowserEdge, nil
	case cookies.BrowserFirefox:
		return libsweetcookie.BrowserFirefox, nil
	case cookies.BrowserFloorp:
		return libsweetcookie.BrowserFloorp, nil
	case cookies.BrowserHelium:
		return libsweetcookie.BrowserHelium, nil
	case cookies.BrowserLibreWolf:
		return libsweetcookie.BrowserLibreWolf, nil
	case cookies.BrowserOpera:
		return libsweetcookie.BrowserOpera, nil
	case cookies.BrowserSafari:
		return libsweetcookie.BrowserSafari, nil
	case cookies.BrowserVivaldi:
		return libsweetcookie.BrowserVivaldi, nil
	case cookies.BrowserWaterfox:
		return libsweetcookie.BrowserWaterfox, nil
	case cookies.BrowserWhale:
		return libsweetcookie.BrowserWhale, nil
	case cookies.BrowserZen:
		return libsweetcookie.BrowserZen, nil
	default:
		return "", fmt.Errorf("unsupported sweetcookie browser %s", browser)
	}
}
