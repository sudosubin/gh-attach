package ghweb

import (
	"errors"
	"fmt"
	"net/http"
)

const maxRedirects = 10

var authHeaders = []string{"Authorization", "GitHub-Remote-Auth"}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("ghweb: stopped after 10 redirects")
	}

	firstHost := via[0].URL.Host
	if via[0].URL.Scheme == "https" && req.URL.Scheme != "https" {
		return fmt.Errorf("ghweb: refusing to follow redirect from https to %s", req.URL.Scheme)
	}

	req.Header.Del("Cookie")
	if crossedHost(firstHost, req.URL.Host, via) {
		for _, h := range authHeaders {
			req.Header.Del(h)
		}
	}
	return c.attachCookie(req)
}

func crossedHost(firstHost, dest string, via []*http.Request) bool {
	if firstHost != dest {
		return true
	}
	for _, r := range via[1:] {
		if r.URL.Host != firstHost {
			return true
		}
	}
	return false
}
