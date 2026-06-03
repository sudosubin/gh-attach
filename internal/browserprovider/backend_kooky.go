package browserprovider

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
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

type storeContext struct {
	browser cookies.Browser
	path    string
	profile string
}

func (b *kookyBackend) Load(ctx context.Context, host string, source cookies.Source) ([]CookieSet, error) {
	if source.Browser != cookies.BrowserFirefox && source.Browser != cookies.BrowserSafari {
		return nil, fmt.Errorf("kooky backend supports only firefox and safari, got %s", source.Browser)
	}
	sel := cookies.ParseProfileSelector(source.Profile)

	if source.CookieStorePath != "" {
		sc := storeContext{browser: source.Browser, path: source.CookieStorePath}
		sets, err := b.loadStore(ctx, host, sc, sel)
		if err != nil {
			return nil, err
		}
		if len(sets) == 0 {
			return nil, fmt.Errorf("no cookies found")
		}
		return sets, nil
	}

	var (
		all         []CookieSet
		storeErrors []string
	)
	for store, err := range rootkooky.TraverseCookieStores(ctx) {
		if err != nil {
			if store != nil {
				store.Close()
			}
			continue
		}
		if store == nil {
			continue
		}

		storeBrowser := store.Browser()
		storeProfile := store.Profile()
		storePath := store.FilePath()
		store.Close()

		if storeBrowser != string(source.Browser) {
			continue
		}
		if sel.Profile != "" && !profileMatches(sel.Profile, storeProfile) {
			continue
		}

		sc := storeContext{browser: source.Browser, path: storePath, profile: storeProfile}
		sets, err := b.loadStore(ctx, host, sc, sel)
		if err != nil {
			storeErrors = append(storeErrors, fmt.Sprintf("%s: %v", storePath, err))
			continue
		}
		all = append(all, sets...)
	}

	if len(all) == 0 && len(storeErrors) == 0 {
		return nil, fmt.Errorf("no cookies found")
	}
	if len(all) == 0 && len(storeErrors) > 0 {
		return nil, fmt.Errorf("no cookies found (%d store(s) failed: %s)", len(storeErrors), strings.Join(storeErrors, "; "))
	}

	return all, nil
}

func (b *kookyBackend) loadStore(ctx context.Context, host string, sc storeContext, sel cookies.ProfileSelector) ([]CookieSet, error) {
	filters := []rootkooky.Filter{
		rootkooky.Valid,
		rootkooky.DomainHasSuffix(host),
	}

	var (
		cookiesOut []*rootkooky.Cookie
		err        error
	)
	switch sc.browser {
	case cookies.BrowserFirefox:
		cookiesOut, err = kfirefox.ReadCookies(ctx, sc.path, filters...)
	case cookies.BrowserSafari:
		cookiesOut, err = ksafari.ReadCookies(ctx, sc.path, filters...)
	default:
		return nil, fmt.Errorf("unsupported browser %s", sc.browser)
	}
	if err != nil {
		return nil, err
	}

	groups := map[kookyGroupKey][]*http.Cookie{}
	for _, ck := range cookiesOut {
		if ck == nil {
			continue
		}
		// Cookie-level metadata wins over the store default when present.
		profile := sc.profile
		if ck.Browser != nil {
			profile = ck.Browser.Profile()
		}
		id, name := splitContainer(ck.Container)
		key, ok := cookieGroupKey(sel, profile, id, name)
		if !ok {
			continue
		}
		hc := ck.Cookie
		groups[key] = append(groups[key], &hc)
	}

	return finalizeKookyGroups(groups), nil
}

type kookyGroupKey struct {
	Profile       string
	ContainerID   string
	ContainerName string
}

// kooky's Container field is "N|Name", "N", or "".
func splitContainer(raw string) (id, name string) {
	id, name, _ = strings.Cut(raw, "|")
	return id, name
}

func cookieGroupKey(sel cookies.ProfileSelector, profile string, id, name string) (kookyGroupKey, bool) {
	if sel.Profile != "" && !profileMatches(sel.Profile, profile) {
		return kookyGroupKey{}, false
	}
	if sel.Container != "" && !containerMatches(sel, id, name) {
		return kookyGroupKey{}, false
	}
	return kookyGroupKey{Profile: profile, ContainerID: id, ContainerName: name}, true
}

func containerMatches(sel cookies.ProfileSelector, id, name string) bool {
	if sel.MatchByID {
		return id != "" && strings.EqualFold(sel.Container, id)
	}
	return name != "" && strings.EqualFold(sel.Container, name)
}

func finalizeKookyGroups(groups map[kookyGroupKey][]*http.Cookie) []CookieSet {
	keys := slices.Collect(maps.Keys(groups))
	slices.SortFunc(keys, func(a, b kookyGroupKey) int {
		return cmp.Or(
			cmp.Compare(a.Profile, b.Profile),
			compareContainerID(a.ContainerID, b.ContainerID),
			cmp.Compare(a.ContainerName, b.ContainerName),
		)
	})

	sets := make([]CookieSet, 0, len(keys))
	for _, k := range keys {
		var profile string
		switch {
		case k.ContainerID == "" && k.ContainerName == "":
			profile = k.Profile
		case k.ContainerName != "":
			profile = cookies.FormatProfileSelector(k.Profile, k.ContainerName, false)
		default:
			profile = cookies.FormatProfileSelector(k.Profile, k.ContainerID, true)
		}
		sets = append(sets, CookieSet{Profile: profile, Cookies: groups[k]})
	}
	return sets
}

// Callers always guard with `expected != ""`, so empty input is not handled here.
func profileMatches(expected, actual string) bool {
	return strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(actual))
}

// Firefox userContextId is numeric ("1", "2", ...); fall back to lex compare if not.
func compareContainerID(a, b string) int {
	aN, aErr := strconv.Atoi(a)
	bN, bErr := strconv.Atoi(b)
	if aErr == nil && bErr == nil {
		return cmp.Compare(aN, bN)
	}
	return cmp.Compare(a, b)
}
