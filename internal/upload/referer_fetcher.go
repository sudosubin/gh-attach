package upload

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// RefererPage is a GitHub page whose URL and CSRF tokens are used as the upload referer context.
type RefererPage struct {
	URL  string
	Body []byte
}

type RefererPageFetcher interface {
	Fetch(ctx context.Context, client *http.Client, cookieHeader string, userAgent string) (*RefererPage, error)
}

type LatestCommitSHAResolver interface {
	LatestCommitSHA(owner string, name string) (string, error)
}

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

func NewLatestCommitPageFetcher(host string, owner string, name string, resolver LatestCommitSHAResolver) RefererPageFetcher {
	return latestCommitPageFetcher{host: host, owner: owner, name: name, resolver: resolver}
}

type latestCommitPageFetcher struct {
	host     string
	owner    string
	name     string
	resolver LatestCommitSHAResolver
}

func (f latestCommitPageFetcher) Fetch(ctx context.Context, client *http.Client, cookieHeader string, userAgent string) (*RefererPage, error) {
	if f.resolver == nil {
		return nil, fmt.Errorf("latest commit resolver is required")
	}

	sha, err := f.resolver.LatestCommitSHA(f.owner, f.name)
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
