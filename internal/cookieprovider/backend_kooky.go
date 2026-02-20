package cookieprovider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	rootkooky "github.com/browserutils/kooky"
	kfirefox "github.com/browserutils/kooky/browser/firefox"
	ksafari "github.com/browserutils/kooky/browser/safari"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type kookyBackend struct{}

func newKookyBackend() *kookyBackend {
	return &kookyBackend{}
}

func (b *kookyBackend) Name() string {
	return "kooky"
}

func (b *kookyBackend) Load(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error) {
	if source.Browser != cookies.BrowserFirefox && source.Browser != cookies.BrowserSafari {
		return nil, fmt.Errorf("kooky backend supports only firefox and safari, got %s", source.Browser)
	}

	if source.CookieStorePath != "" {
		return b.loadFromExplicitPath(ctx, host, source)
	}

	filters := []rootkooky.Filter{
		rootkooky.Valid,
		rootkooky.DomainHasSuffix(host),
	}

	seq := rootkooky.TraverseCookies(ctx, filters...).OnlyCookies()
	out := make([]*http.Cookie, 0)
	for ck, err := range seq {
		if err != nil || ck == nil {
			continue
		}
		if ck.Browser == nil {
			continue
		}
		if ck.Browser.Browser() != string(source.Browser) {
			continue
		}
		if source.Profile != "" && !profileMatches(source.Profile, ck.Browser.Profile()) {
			continue
		}

		hc := ck.Cookie
		out = append(out, &hc)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no cookies found")
	}
	return out, nil
}

func (b *kookyBackend) loadFromExplicitPath(ctx context.Context, host string, source cookies.Source) ([]*http.Cookie, error) {
	filters := []rootkooky.Filter{
		rootkooky.Valid,
		rootkooky.DomainHasSuffix(host),
	}

	var (
		cookiesOut []*rootkooky.Cookie
		err        error
	)

	switch source.Browser {
	case cookies.BrowserFirefox:
		cookiesOut, err = kfirefox.ReadCookies(ctx, source.CookieStorePath, filters...)
	case cookies.BrowserSafari:
		cookiesOut, err = ksafari.ReadCookies(ctx, source.CookieStorePath, filters...)
	default:
		return nil, fmt.Errorf("unsupported explicit-path browser %s", source.Browser)
	}
	if err != nil {
		return nil, err
	}

	out := make([]*http.Cookie, 0, len(cookiesOut))
	for _, ck := range cookiesOut {
		if ck == nil {
			continue
		}
		if source.Profile != "" && ck.Browser != nil && !profileMatches(source.Profile, ck.Browser.Profile()) {
			continue
		}
		hc := ck.Cookie
		out = append(out, &hc)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no cookies found")
	}
	return out, nil
}

func profileMatches(expected string, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" {
		return true
	}
	if expected == actual {
		return true
	}
	return strings.EqualFold(expected, actual)
}
