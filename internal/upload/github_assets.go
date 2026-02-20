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
	"regexp"
	"strconv"
	"strings"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
)

type Asset struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	ContentType  string  `json:"content_type"`
	Href         string  `json:"href"`
	OriginalName *string `json:"original_name"`
}

type policiesResponse struct {
	UploadURL                    string            `json:"upload_url"`
	UploadAuthenticityToken      string            `json:"upload_authenticity_token"`
	Form                         map[string]string `json:"form"`
	Header                       map[string]string `json:"header"`
	Asset                        Asset             `json:"asset"`
	AssetUploadURL               string            `json:"asset_upload_url"`
	AssetUploadAuthenticityToken string            `json:"asset_upload_authenticity_token"`
	SameOrigin                   bool              `json:"same_origin"`
}

var (
	authTokenInputPattern = regexp.MustCompile(`name=["']authenticity_token["'][^>]*value=["']([^"']+)["']`)
	csrfMetaPattern       = regexp.MustCompile(`<meta[^>]*name=["']csrf-token["'][^>]*content=["']([^"']+)["']`)
)

func UploadPoliciesAsset(ctx context.Context, host string, repoFullName string, repositoryID int64, filePath string, session browserprovider.BrowserSession) (Asset, error) {
	if repositoryID <= 0 {
		return Asset{}, fmt.Errorf("invalid repository id")
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

	client := &http.Client{}
	baseURL := fmt.Sprintf("https://%s", host)
	issueNewURL := fmt.Sprintf("%s/%s/issues/new", baseURL, repoFullName)
	userAgent, err := browserprovider.UserAgentForBrowser(session.Browser)
	if err != nil {
		return Asset{}, err
	}
	if strings.TrimSpace(session.UserAgent) != "" && session.UserAgent != userAgent {
		return Asset{}, fmt.Errorf("browser session user-agent mismatch for browser %s", session.Browser)
	}

	cookieHeader, err := cookieHeaderForURL(session.Cookies, issueNewURL)
	if err != nil {
		return Asset{}, err
	}

	authenticityToken, _ := fetchAuthenticityToken(ctx, client, issueNewURL, cookieHeader, userAgent)

	policies, err := requestPolicies(ctx, client, baseURL, cookieHeader, repositoryID, fileName, fileInfo.Size(), contentType, authenticityToken, userAgent)
	if err != nil {
		return Asset{}, err
	}

	if err := uploadBinary(ctx, client, filePath, contentType, cookieHeader, policies, userAgent); err != nil {
		return Asset{}, err
	}

	asset, err := finalizeAsset(ctx, client, baseURL, cookieHeader, policies, userAgent)
	if err != nil {
		if policies.Asset.Href != "" {
			return policies.Asset, nil
		}
		return Asset{}, err
	}

	return asset, nil
}

func fetchAuthenticityToken(ctx context.Context, client *http.Client, pageURL string, cookieHeader string, userAgent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	setDefaultHeaders(req, pageURL)
	if cookieHeader != "" {
		setCookieAndUserAgent(req, cookieHeader, userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed to fetch authenticity token page: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	text := string(body)

	if m := authTokenInputPattern.FindStringSubmatch(text); len(m) > 1 {
		return m[1], nil
	}
	if m := csrfMetaPattern.FindStringSubmatch(text); len(m) > 1 {
		return m[1], nil
	}

	return "", fmt.Errorf("authenticity token not found")
}

func requestPolicies(ctx context.Context, client *http.Client, baseURL string, cookieHeader string, repositoryID int64, fileName string, fileSize int64, contentType string, authenticityToken string, userAgent string) (policiesResponse, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("repository_id", strconv.FormatInt(repositoryID, 10))
	_ = writer.WriteField("name", fileName)
	_ = writer.WriteField("size", strconv.FormatInt(fileSize, 10))
	_ = writer.WriteField("content_type", contentType)
	if authenticityToken != "" {
		_ = writer.WriteField("authenticity_token", authenticityToken)
	}
	if err := writer.Close(); err != nil {
		return policiesResponse{}, err
	}

	url := baseURL + "/upload/policies/assets"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, payload)
	if err != nil {
		return policiesResponse{}, err
	}
	setDefaultHeaders(req, baseURL+"/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("GitHub-Verified-Fetch", "true")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if cookieHeader != "" {
		setCookieAndUserAgent(req, cookieHeader, userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return policiesResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return policiesResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return policiesResponse{}, fmt.Errorf("policies request failed: %s: %s", resp.Status, string(body))
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

func uploadBinary(ctx context.Context, client *http.Client, filePath string, contentType string, cookieHeader string, policies policiesResponse, userAgent string) error {
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
	if cookieHeader != "" {
		setCookieAndUserAgent(req, cookieHeader, userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("binary upload failed: %s: %s", resp.Status, string(body))
	}
	return nil
}

func finalizeAsset(ctx context.Context, client *http.Client, baseURL string, cookieHeader string, policies policiesResponse, userAgent string) (Asset, error) {
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
		finalURL = baseURL + finalURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, finalURL, payload)
	if err != nil {
		return Asset{}, err
	}
	setDefaultHeaders(req, baseURL+"/")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if cookieHeader != "" {
		setCookieAndUserAgent(req, cookieHeader, userAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Asset{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Asset{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Asset{}, fmt.Errorf("asset finalize failed: %s: %s", resp.Status, string(body))
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
