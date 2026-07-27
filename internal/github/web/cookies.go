package web

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

type scopedCookieJar struct {
	jar   http.CookieJar
	hosts map[string]struct{}
}

func newScopedCookieJar(cookies []*http.Cookie) http.CookieJar {
	jar, _ := cookiejar.New(nil)
	scoped := &scopedCookieJar{jar: jar, hosts: map[string]struct{}{}}
	for _, c := range cookies {
		if host := strings.ToLower(strings.TrimPrefix(c.Domain, ".")); host != "" {
			scoped.hosts[host] = struct{}{}
		}
	}
	for _, c := range cookies {
		host := strings.TrimPrefix(c.Domain, ".")
		if host != "" {
			scoped.jar.SetCookies(&url.URL{Scheme: "https", Host: host, Path: "/"}, []*http.Cookie{c})
		}
	}
	return scoped
}

func (j *scopedCookieJar) Cookies(u *url.URL) []*http.Cookie {
	if _, ok := j.hosts[strings.ToLower(u.Hostname())]; !ok {
		return nil
	}
	return j.jar.Cookies(u)
}

func (j *scopedCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	if _, ok := j.hosts[strings.ToLower(u.Hostname())]; ok {
		j.jar.SetCookies(u, cookies)
	}
}
