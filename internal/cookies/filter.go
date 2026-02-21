package cookies

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// ValuesForHost returns distinct non-empty values of cookies with the given
// name that are applicable to the given host, respecting cookie domain rules.
func ValuesForHost(in []*http.Cookie, name string, host string) []string {
	host = strings.TrimSpace(host)
	if host == "" {
		return valuesByName(in, name)
	}

	u := &url.URL{Scheme: "https", Host: host, Path: "/"}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return valuesByName(in, name)
	}
	jar.SetCookies(u, in)

	return valuesByName(jar.Cookies(u), name)
}

func valuesByName(in []*http.Cookie, name string) []string {
	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, c := range in {
		if c == nil || c.Name != name || c.Value == "" {
			continue
		}
		if _, ok := seen[c.Value]; ok {
			continue
		}
		seen[c.Value] = struct{}{}
		out = append(out, c.Value)
	}
	return out
}
