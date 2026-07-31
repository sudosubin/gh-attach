package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/rest"
	"github.com/sudosubin/gh-attach/internal/github/web"
)

type DownloadRequest struct {
	URL             string
	Browser         string
	Profile         string
	CookieStorePath string
	SessionToken    string
	Verbose         bool
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
		session, sessionErr := s.resolveBrowserSession(ctx, host, req)
		if sessionErr != nil {
			return nil, sessionErr
		}
		return s.download(ctx, attachmentURL, session, "")
	}

	token, _ := auth.TokenForHost(host)
	body, status, downloadErr := s.downloadResponse(ctx, attachmentURL, web.Session{UserAgent: "gh-attach"}, token)
	if downloadErr == nil {
		return body, nil
	}
	if !shouldRetryWithCookies(status) {
		return nil, downloadErr
	}
	return s.retryWithBrowserCookies(ctx, attachmentURL, host, req, downloadErr)
}

func parseAttachmentURL(rawURL string) (string, string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", fmt.Errorf("invalid attachment URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.User != nil || parsed.Fragment != "" {
		return "", "", errors.New("attachment URL must be an HTTPS URL without credentials or a fragment")
	}

	host := strings.ToLower(parsed.Hostname())
	known := host == "github.com" || slices.ContainsFunc(auth.KnownHosts(), func(candidate string) bool {
		return strings.EqualFold(candidate, host)
	})
	if !known {
		return "", "", fmt.Errorf("attachment URL host %q is not configured in gh", host)
	}

	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	validAsset := len(segments) == 3 && segments[0] == "user-attachments" && segments[1] == "assets" && segments[2] != ""
	validFile := len(segments) >= 4 && segments[0] == "user-attachments" && segments[1] == "files" &&
		segments[2] != "" && segments[3] != ""
	if !validAsset && !validFile {
		return "", "", errors.New("URL is not a GitHub user-attachment")
	}

	return parsed.String(), host, nil
}

func (s *Service) retryWithBrowserCookies(
	ctx context.Context,
	attachmentURL string,
	host string,
	req DownloadRequest,
	firstErr error,
) (io.ReadCloser, error) {
	session, err := s.resolveBrowserSession(ctx, host, req)
	if err != nil {
		return nil, errors.Join(firstErr, fmt.Errorf("browser cookie fallback: %w", err))
	}
	body, err := s.download(ctx, attachmentURL, session, "")
	if err != nil {
		return nil, errors.Join(firstErr, fmt.Errorf("browser cookie fallback: %w", err))
	}
	return body, nil
}

func (s *Service) resolveBrowserSession(ctx context.Context, host string, req DownloadRequest) (web.Session, error) {
	sources, err := cookies.ResolveSources(cookies.ResolveInput{
		Browser:         req.Browser,
		Profile:         req.Profile,
		CookieStorePath: req.CookieStorePath,
	})
	if err != nil {
		return web.Session{}, err
	}
	loginResolver := s.loginResolver
	if loginResolver == nil {
		apiService, err := rest.NewService(host, nil)
		if err != nil {
			return web.Session{}, err
		}
		loginResolver = NewConfigLoginResolver(apiService)
	}
	resolved, err := NewSessionResolver(
		loginResolver,
		NewCookieResolver(s.providers, req.Verbose, s.stderr),
	).Resolve(ctx, host, sources)
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
	body, _, err := s.downloadResponse(ctx, attachmentURL, session, token)
	return body, err
}

func (s *Service) downloadResponse(
	ctx context.Context,
	attachmentURL string,
	session web.Session,
	token string,
) (io.ReadCloser, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachmentURL, nil)
	if err != nil {
		return nil, 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := web.NewClient(s.httpClient, session.UserAgent, session.Cookies).Do(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.StatusCode == http.StatusOK {
		return resp.Body, resp.StatusCode, nil
	}

	status := resp.StatusCode
	err = api.HandleHTTPError(resp)
	_ = resp.Body.Close()
	return nil, status, err
}

func shouldRetryWithCookies(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound
}
