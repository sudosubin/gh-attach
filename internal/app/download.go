package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/web"
)

const bearerUserAgent = "gh-attach"

var errDownloadAuthenticationRequired = errors.New("GitHub returned a sign-in or SSO page; sign in to GitHub and complete SSO in your browser, then retry")

var (
	authenticationBodyPattern = regexp.MustCompile(`(?i)<body\b[^>]*\sclass\s*=\s*["'][^"']*\bsession-authentication\b[^"']*["']`)
	authenticationFormPattern = regexp.MustCompile(`(?i)<form\b[^>]*\saction\s*=\s*["'](?:https://[^/"']+)?/(?:sessions?|login|(?:orgs|enterprises)/[^/"'?#]+/(?:sso|saml/initiate))(?:\?[^"']*)?["']`)
)

type DownloadRequest struct {
	URL             string
	Browser         string
	Profile         string
	CookieStorePath string
	SessionToken    string
	Verbose         bool
}

func (req DownloadRequest) cookieSources() cookies.ResolveInput {
	return cookies.ResolveInput{
		Browser:         req.Browser,
		Profile:         req.Profile,
		CookieStorePath: req.CookieStorePath,
	}
}

func (s *Service) Download(ctx context.Context, req DownloadRequest) (io.ReadCloser, error) {
	attachmentURL, host, err := parseAttachmentURL(req.URL)
	if err != nil {
		return nil, err
	}

	if req.SessionToken != "" {
		session, sessionErr := newTokenSession(host, req.SessionToken)
		if sessionErr != nil {
			return nil, sessionErr
		}
		return s.download(ctx, attachmentURL, session, "")
	}

	if req.Browser != "" || req.Profile != "" || req.CookieStorePath != "" {
		session, sessionErr := s.resolveBrowserSession(ctx, host, req.cookieSources(), req.Verbose)
		if sessionErr != nil {
			return nil, sessionErr
		}
		return s.download(ctx, attachmentURL, session, "")
	}

	token, _ := auth.TokenForHost(host)
	body, downloadErr := s.download(ctx, attachmentURL, web.Session{UserAgent: bearerUserAgent}, token)
	if downloadErr == nil {
		return body, nil
	}
	if !shouldRetryWithCookies(downloadErr) {
		return nil, downloadErr
	}
	return s.retryWithBrowserCookies(ctx, attachmentURL, host, req, downloadErr)
}

// parseAttachmentURL validates a user-attachment URL and returns it with the configured host to authenticate as.
func parseAttachmentURL(rawURL string) (string, string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", fmt.Errorf("invalid attachment URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", errors.New("attachment URL must be an HTTPS URL without credentials, query, or fragment")
	}

	host, known := knownAuthHost(parsed.Host)
	if !known {
		return "", "", fmt.Errorf("attachment URL host %q is not configured in gh", parsed.Host)
	}

	segments := strings.Split(strings.TrimPrefix(parsed.EscapedPath(), "/"), "/")
	validAsset := len(segments) == 3 && segments[0] == "user-attachments" && segments[1] == "assets" && segments[2] != ""
	validFile := len(segments) >= 4 && segments[0] == "user-attachments" && segments[1] == "files" &&
		segments[2] != "" && segments[3] != ""
	if (!validAsset && !validFile) || segments[len(segments)-1] == "" {
		return "", "", errors.New("URL is not a GitHub user-attachment")
	}

	return parsed.String(), host, nil
}

// knownAuthHost matches the full URL authority against gh hosts, allowing subdomains for GHES subdomain isolation.
func knownAuthHost(urlHost string) (string, bool) {
	urlHost = strings.ToLower(urlHost)
	for _, candidate := range append(auth.KnownHosts(), "github.com") {
		candidate = strings.ToLower(candidate)
		if urlHost == candidate || strings.HasSuffix(urlHost, "."+candidate) {
			return candidate, true
		}
	}
	return "", false
}

func (s *Service) retryWithBrowserCookies(
	ctx context.Context,
	attachmentURL string,
	host string,
	req DownloadRequest,
	firstErr error,
) (io.ReadCloser, error) {
	session, err := s.resolveBrowserSession(ctx, host, req.cookieSources(), req.Verbose)
	if err != nil {
		return nil, errors.Join(firstErr, fmt.Errorf("browser cookie fallback: %w", err))
	}
	body, err := s.download(ctx, attachmentURL, session, "")
	if err != nil {
		return nil, errors.Join(firstErr, fmt.Errorf("browser cookie fallback: %w", err))
	}
	return body, nil
}

func (s *Service) resolveBrowserSession(
	ctx context.Context,
	host string,
	input cookies.ResolveInput,
	verbose bool,
) (web.Session, error) {
	sources, err := cookies.ResolveSources(input)
	if err != nil {
		return web.Session{}, err
	}
	loginResolver := s.loginResolver
	if loginResolver == nil {
		loginResolver = NewConfigLoginResolver(lazyAPILoginResolver{host: host})
	}
	resolved, err := NewSessionResolver(
		loginResolver,
		NewCookieResolver(s.providers, verbose, s.stderr),
	).Resolve(ctx, (&url.URL{Host: host}).Hostname(), sources)
	if err != nil {
		return web.Session{}, err
	}
	return web.Session{
		Cookies:   resolved.Session.Cookies,
		UserAgent: resolved.Session.UserAgent,
	}, nil
}

func (s *Service) download(
	ctx context.Context,
	attachmentURL string,
	session web.Session,
	token string,
) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachmentURL, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := web.NewClient(s.httpClient, session.UserAgent, session.Cookies).Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		if err := checkDownloadAuthentication(resp); err != nil {
			_ = resp.Body.Close()
			return nil, err
		}
		return resp.Body, nil
	}

	err = api.HandleHTTPError(resp)
	_ = resp.Body.Close()
	return nil, err
}

func checkDownloadAuthentication(resp *http.Response) error {
	contentType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	disposition, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if (contentType != "text/html" && contentType != "application/xhtml+xml") || disposition == "attachment" {
		return nil
	}
	if _, known := knownAuthHost(resp.Request.URL.Host); !known {
		return nil
	}

	// GitHub can serve a sign-in or SSO page as HTTP 200 at the attachment URL.
	const prefixSize = 64 * 1024
	reader := bufio.NewReaderSize(resp.Body, prefixSize)
	prefix, err := reader.Peek(prefixSize)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("inspect download response: %w", err)
	}
	if authenticationBodyPattern.Match(prefix) && authenticationFormPattern.Match(prefix) {
		return errDownloadAuthenticationRequired
	}
	resp.Body = struct {
		io.Reader
		io.Closer
	}{Reader: reader, Closer: resp.Body}
	return nil
}

func shouldRetryWithCookies(err error) bool {
	if errors.Is(err, errDownloadAuthenticationRequired) {
		return true
	}
	if httpErr, ok := errors.AsType[*api.HTTPError](err); ok {
		return httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode == http.StatusForbidden || httpErr.StatusCode == http.StatusNotFound
	}
	return false
}
