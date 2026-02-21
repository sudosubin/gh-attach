package upload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/ghapi"
)

type RefererPage struct {
	URL  string
	Body []byte
}

type RefererPageFetcher interface {
	Fetch(ctx context.Context, client *http.Client, cookieHeader string, userAgent string) (*RefererPage, error)
}

var latestCommitSHAFn = ghapi.LatestCommitSHA

func NewIssueNewPageFetcher(host string, repoFullName string) RefererPageFetcher {
	return issueNewPageFetcher{host: host, repoFullName: repoFullName}
}

type issueNewPageFetcher struct {
	host         string
	repoFullName string
}

func (f issueNewPageFetcher) Fetch(ctx context.Context, client *http.Client, cookieHeader string, userAgent string) (*RefererPage, error) {
	pageURL := fmt.Sprintf("https://%s/%s/issues/new", f.host, f.repoFullName)
	body, err := fetchPageBody(ctx, client, pageURL, cookieHeader, userAgent)
	if err != nil || body == nil {
		return nil, err
	}

	return &RefererPage{URL: pageURL, Body: body}, nil
}

func NewLatestCommitPageFetcher(host string, owner string, name string) RefererPageFetcher {
	return latestCommitPageFetcher{host: host, owner: owner, name: name}
}

type latestCommitPageFetcher struct {
	host  string
	owner string
	name  string
}

func (f latestCommitPageFetcher) Fetch(ctx context.Context, client *http.Client, cookieHeader string, userAgent string) (*RefererPage, error) {
	sha, err := latestCommitSHAFn(f.host, f.owner, f.name)
	if err != nil {
		return nil, err
	}

	pageURL := fmt.Sprintf("https://%s/%s/%s/commit/%s", f.host, f.owner, f.name, sha)
	body, err := fetchPageBody(ctx, client, pageURL, cookieHeader, userAgent)
	if err != nil || body == nil {
		return nil, err
	}

	return &RefererPage{URL: pageURL, Body: body}, nil
}

func ResolveUploadRefererPage(ctx context.Context, host string, fetchers []RefererPageFetcher, session browserprovider.BrowserSession, client *http.Client) (*RefererPage, error) {
	if len(fetchers) == 0 {
		return nil, fmt.Errorf("no referer page fetchers configured")
	}

	userAgent, err := browserprovider.UserAgentForBrowser(session.Browser)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(session.UserAgent) != "" && session.UserAgent != userAgent {
		return nil, fmt.Errorf("browser session user-agent mismatch for browser %s", session.Browser)
	}

	cookieHeader, err := cookieHeaderForURL(session.Cookies, fmt.Sprintf("https://%s/", host))
	if err != nil {
		return nil, err
	}

	if client == nil {
		client = &http.Client{}
	}

	var lastErr error
	for _, fetcher := range fetchers {
		page, err := fetcher.Fetch(ctx, client, cookieHeader, userAgent)
		if err != nil {
			lastErr = err
			continue
		}
		if page != nil {
			return page, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}

	return nil, fmt.Errorf("failed to resolve accessible referer page")
}

func fetchPageBody(ctx context.Context, client *http.Client, pageURL string, cookieHeader string, userAgent string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}
	setDefaultHeaders(req, pageURL)
	if cookieHeader != "" {
		setCookieAndUserAgent(req, cookieHeader, userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
