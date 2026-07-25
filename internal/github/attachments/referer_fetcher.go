package attachments

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/sudosubin/gh-attach/internal/github/web"
)

// RefererPage carries the referer URL and the metadata parsed from its HTML at fetch time.
type RefererPage struct {
	URL  string
	Meta refererPageMetadata
}

type RefererPageFetcher interface {
	Fetch(ctx context.Context, client *web.Client) (*RefererPage, error)
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

func (f issueNewPageFetcher) Fetch(ctx context.Context, client *web.Client) (*RefererPage, error) {
	pageURL := fmt.Sprintf("https://%s/%s/issues/new", f.host, f.repoFullName)
	return fetchAndParse(ctx, client, pageURL)
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

func (f latestCommitPageFetcher) Fetch(ctx context.Context, client *web.Client) (*RefererPage, error) {
	if f.resolver == nil {
		return nil, fmt.Errorf("latest commit resolver is required")
	}

	sha, err := f.resolver.LatestCommitSHA(f.owner, f.name)
	if err != nil {
		return nil, err
	}

	pageURL := fmt.Sprintf("https://%s/%s/%s/commit/%s", f.host, f.owner, f.name, sha)
	return fetchAndParse(ctx, client, pageURL)
}

// fetchAndParse GETs and parses the page; non-200 returns (nil, nil) to try the next fetcher.
func fetchAndParse(ctx context.Context, client *web.Client, pageURL string) (*RefererPage, error) {
	body, status, err := client.Get(ctx, pageURL)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, nil
	}

	html := string(body)
	meta := parseCSRFMetadata(html)
	meta.RepositoryID = extractRepositoryID(html)
	return &RefererPage{URL: pageURL, Meta: meta}, nil
}

type refererPageMetadata struct {
	RepositoryID        int64
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

// parseCSRFMetadata extracts the CSRF context common to every GitHub page.
func parseCSRFMetadata(html string) refererPageMetadata {
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

// Every fetcher tries all patterns, in priority order, since embedding varies by page/rendering.
var repositoryIDPatterns = []*regexp.Regexp{
	regexp.MustCompile(`"repository":\{"id":"[^"]*","databaseId":(\d+)`),
	regexp.MustCompile(`octolytics-dimension-repository_id["'][^>]*content=["'](\d+)["']`),
	regexp.MustCompile(`repository_id=(\d+)`),
}

// extractRepositoryID returns the first positive id matched, or 0 to signal API fallback.
func extractRepositoryID(html string) int64 {
	for _, pattern := range repositoryIDPatterns {
		m := pattern.FindStringSubmatch(html)
		if len(m) <= 1 {
			continue
		}
		id, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		return id
	}
	return 0
}
