package upload

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/sudosubin/gh-attach/internal/ghweb"
)

// RefererPage is a GitHub page whose URL and CSRF tokens are used as the upload referer context.
type RefererPage struct {
	URL  string
	Body []byte
}

type RefererPageFetcher interface {
	Fetch(ctx context.Context, client *ghweb.Client) (*RefererPage, error)
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

func (f issueNewPageFetcher) Fetch(ctx context.Context, client *ghweb.Client) (*RefererPage, error) {
	pageURL := fmt.Sprintf("https://%s/%s/issues/new", f.host, f.repoFullName)
	return fetchRefererPage(ctx, client, pageURL)
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

func (f latestCommitPageFetcher) Fetch(ctx context.Context, client *ghweb.Client) (*RefererPage, error) {
	if f.resolver == nil {
		return nil, fmt.Errorf("latest commit resolver is required")
	}

	sha, err := f.resolver.LatestCommitSHA(f.owner, f.name)
	if err != nil {
		return nil, err
	}

	pageURL := fmt.Sprintf("https://%s/%s/%s/commit/%s", f.host, f.owner, f.name, sha)
	return fetchRefererPage(ctx, client, pageURL)
}

// fetchRefererPage treats non-200 as "try the next fetcher," not an error; only a transport failure is.
func fetchRefererPage(ctx context.Context, client *ghweb.Client, pageURL string) (*RefererPage, error) {
	body, status, err := client.Get(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, nil
	}
	return &RefererPage{URL: pageURL, Body: body}, nil
}

type refererPageMetadata struct {
	AuthenticityToken   string
	FetchNonce          string
	GitHubClientVersion string
}

var (
	authTokenInputPattern          = regexp.MustCompile(`name=["']authenticity_token["'][^>]*value=["']([^"']+)["']`)
	csrfMetaPattern                = regexp.MustCompile(`<meta[^>]*name=["']csrf-token["'][^>]*content=["']([^"']+)["']`)
	fetchNonceMetaPattern          = regexp.MustCompile(`<meta[^>]*name=["']fetch-nonce["'][^>]*content=["']([^"']+)["']`)
	githubClientVersionMetaPattern = regexp.MustCompile(`<meta[^>]*name=["']release["'][^>]*content=["']([^"']+)["']`)
)

func extractRefererPageMetadata(html string) refererPageMetadata {
	meta := refererPageMetadata{}

	if m := authTokenInputPattern.FindStringSubmatch(html); len(m) > 1 {
		meta.AuthenticityToken = m[1]
	} else if m := csrfMetaPattern.FindStringSubmatch(html); len(m) > 1 {
		meta.AuthenticityToken = m[1]
	}
	if m := fetchNonceMetaPattern.FindStringSubmatch(html); len(m) > 1 {
		meta.FetchNonce = m[1]
	}
	if m := githubClientVersionMetaPattern.FindStringSubmatch(html); len(m) > 1 {
		meta.GitHubClientVersion = m[1]
	}

	return meta
}
