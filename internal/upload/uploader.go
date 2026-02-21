package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
	"github.com/sudosubin/gh-attach/internal/browserprovider"
)

// Uploader carries the shared session context for GitHub user-attachment uploads.
type Uploader struct {
	baseURL      string // e.g. "https://github.com"
	repositoryID int64
	client       *http.Client
	userAgent    string
	cookieHeader string
}

func NewUploader(host string, repositoryID int64, session browserprovider.BrowserSession, client *http.Client) (*Uploader, error) {
	if repositoryID <= 0 {
		return nil, fmt.Errorf("invalid repository id")
	}

	userAgent, err := browserprovider.UserAgentForBrowser(session.Browser)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(session.UserAgent) != "" && session.UserAgent != userAgent {
		return nil, fmt.Errorf("browser session user-agent mismatch for browser %s", session.Browser)
	}

	cookieHeader, err := cookieHeaderForURL(session.Cookies, "https://"+host+"/")
	if err != nil {
		return nil, err
	}

	if client == nil {
		client = &http.Client{}
	}

	return &Uploader{
		baseURL:      "https://" + host,
		repositoryID: repositoryID,
		client:       client,
		userAgent:    userAgent,
		cookieHeader: cookieHeader,
	}, nil
}

func (u *Uploader) ResolveRefererPage(ctx context.Context, fetchers []RefererPageFetcher) (*RefererPage, error) {
	if len(fetchers) == 0 {
		return nil, fmt.Errorf("no referer page fetchers configured")
	}

	var lastErr error
	for _, fetcher := range fetchers {
		page, err := fetcher.Fetch(ctx, u.client, u.cookieHeader, u.userAgent)
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

func (u *Uploader) Upload(ctx context.Context, filePath string, refererPage *RefererPage) (Asset, error) {
	if refererPage == nil || strings.TrimSpace(refererPage.URL) == "" {
		return Asset{}, fmt.Errorf("invalid referer page")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return Asset{}, err
	}
	if fileInfo.IsDir() {
		return Asset{}, fmt.Errorf("%s is a directory", filePath)
	}

	fileName := filepath.Base(filePath)
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	policies, err := u.requestPolicies(ctx, refererPage, fileName, fileInfo.Size(), contentType)
	if err != nil {
		return Asset{}, err
	}

	if err := u.uploadBinary(ctx, filePath, contentType, policies); err != nil {
		return Asset{}, err
	}

	asset, err := u.finalizeAsset(ctx, policies)
	if err != nil {
		if policies.Asset.Href != "" {
			return policies.Asset, nil
		}
		return Asset{}, err
	}

	return asset, nil
}

func (u *Uploader) requestPolicies(ctx context.Context, refererPage *RefererPage, fileName string, fileSize int64, contentType string) (policiesResponse, error) {
	refererMeta := extractRefererPageMetadata(string(refererPage.Body))

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("repository_id", strconv.FormatInt(u.repositoryID, 10))
	_ = writer.WriteField("name", fileName)
	_ = writer.WriteField("size", strconv.FormatInt(fileSize, 10))
	_ = writer.WriteField("content_type", contentType)
	if refererMeta.AuthenticityToken != "" {
		_ = writer.WriteField("authenticity_token", refererMeta.AuthenticityToken)
	}
	if err := writer.Close(); err != nil {
		return policiesResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.baseURL+"/upload/policies/assets", payload)
	if err != nil {
		return policiesResponse{}, err
	}
	setDefaultHeaders(req, refererPage.URL)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("GitHub-Verified-Fetch", "true")
	if strings.TrimSpace(refererMeta.FetchNonce) != "" {
		req.Header.Set("X-Fetch-Nonce", refererMeta.FetchNonce)
	}
	if strings.TrimSpace(refererMeta.GitHubClientVersion) != "" {
		req.Header.Set("X-GitHub-Client-Version", refererMeta.GitHubClientVersion)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if u.cookieHeader != "" {
		setCookieAndUserAgent(req, u.cookieHeader, u.userAgent)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return policiesResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return policiesResponse{}, ghapi.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return policiesResponse{}, err
	}

	var out policiesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return policiesResponse{}, err
	}
	if out.UploadURL == "" {
		return policiesResponse{}, fmt.Errorf("policies response missing upload_url")
	}
	return out, nil
}

func (u *Uploader) uploadBinary(ctx context.Context, filePath string, contentType string, policies policiesResponse) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	for k, v := range policies.Form {
		_ = writer.WriteField(k, v)
	}

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, policies.UploadURL, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if contentType != "" {
		req.Header.Set("X-File-Content-Type", contentType)
	}
	for k, v := range policies.Header {
		req.Header.Set(k, v)
	}
	if policies.SameOrigin && policies.UploadAuthenticityToken != "" {
		req.Header.Set("authenticity_token", policies.UploadAuthenticityToken)
	}
	if u.cookieHeader != "" {
		setCookieAndUserAgent(req, u.cookieHeader, u.userAgent)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ghapi.HandleHTTPError(resp)
	}
	return nil
}

func (u *Uploader) finalizeAsset(ctx context.Context, policies policiesResponse) (Asset, error) {
	if policies.AssetUploadURL == "" {
		return policies.Asset, nil
	}

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("authenticity_token", policies.AssetUploadAuthenticityToken)
	if err := writer.Close(); err != nil {
		return Asset{}, err
	}

	finalURL := policies.AssetUploadURL
	if strings.HasPrefix(finalURL, "/") {
		finalURL = u.baseURL + finalURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, finalURL, payload)
	if err != nil {
		return Asset{}, err
	}
	setDefaultHeaders(req, u.baseURL+"/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if u.cookieHeader != "" {
		setCookieAndUserAgent(req, u.cookieHeader, u.userAgent)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return Asset{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Asset{}, ghapi.HandleHTTPError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Asset{}, err
	}

	var asset Asset
	if err := json.Unmarshal(body, &asset); err != nil {
		return policies.Asset, nil
	}
	if asset.Href == "" && policies.Asset.Href != "" {
		return policies.Asset, nil
	}
	return asset, nil
}

func cookieHeaderForURL(cookies []*http.Cookie, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	jar.SetCookies(u, cookies)

	pairs := make([]string, 0)
	for _, c := range jar.Cookies(u) {
		pairs = append(pairs, c.Name+"="+c.Value)
	}
	return strings.Join(pairs, "; "), nil
}

func setDefaultHeaders(req *http.Request, referer string) {
	req.Header.Set("Origin", req.URL.Scheme+"://"+req.URL.Host)
	req.Header.Set("Referer", referer)
}

func setCookieAndUserAgent(req *http.Request, cookieHeader string, userAgent string) {
	req.Header.Set("Cookie", cookieHeader)
	req.Header.Set("User-Agent", userAgent)
}
