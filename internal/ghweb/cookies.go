package ghweb

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func cookieHeaderForURL(cookies []*http.Cookie, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Collapse duplicate (Name, Domain, Path) keys last-wins; Firefox folds dFPI/private-browsing cookies into the default container.
	type cookieKey struct{ name, domain, path string }
	index := map[cookieKey]int{}
	deduped := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		key := cookieKey{c.Name, c.Domain, c.Path}
		if i, ok := index[key]; ok {
			deduped[i] = c
			continue
		}
		index[key] = len(deduped)
		deduped = append(deduped, c)
	}

	// GitHub cookies are host-only in practice; cookiejar treats any non-empty Domain as subdomain-eligible, so cookies are pre-filtered to an exact host match first.
	host := u.Hostname()
	scoped := make([]*http.Cookie, 0, len(deduped))
	for _, c := range deduped {
		if strings.EqualFold(strings.TrimPrefix(c.Domain, "."), host) {
			scoped = append(scoped, c)
		}
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	jar.SetCookies(u, scoped)

	pairs := make([]string, 0, len(scoped))
	for _, c := range jar.Cookies(u) {
		pairs = append(pairs, c.Name+"="+c.Value)
	}
	return strings.Join(pairs, "; "), nil
}
