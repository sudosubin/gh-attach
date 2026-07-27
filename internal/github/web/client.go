// Package web builds and sends authenticated HTTP requests to a GitHub web host; it knows nothing about upload API shapes — see internal/github/attachments for that.
package web

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client makes GitHub web HTTP calls, scoping cookies to each request's destination host.
type Client struct {
	httpClient *http.Client
	userAgent  string
}

func NewClient(httpClient *http.Client, userAgent string, cookies []*http.Cookie) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	c := &Client{userAgent: userAgent}
	wrapped := *httpClient
	wrapped.CheckRedirect = c.checkRedirect
	wrapped.Jar = newScopedCookieJar(cookies)
	c.httpClient = &wrapped
	return c
}

// Request is a pure description of one multipart HTTP call's shape; DoMultipart turns it into an actual request.
type Request struct {
	Method   string
	URL      string
	Fields   map[string]string
	Headers  map[string]string
	FilePath string // non-empty attaches the file as the "file" form part
	Referer  string // non-empty applies setDefaultHeaders(req, referer)
}

// DoMultipart builds the multipart body, applies headers/cookie/user-agent, and returns the response body.
func (c *Client) DoMultipart(ctx context.Context, req Request) ([]byte, error) {
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	for k, v := range req.Fields {
		_ = writer.WriteField(k, v)
	}
	if req.FilePath != "" {
		file, err := os.Open(req.FilePath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = file.Close() }()

		part, err := writer.CreateFormFile("file", filepath.Base(req.FilePath))
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, payload)
	if err != nil {
		return nil, err
	}
	if req.Referer != "" {
		if err := setDefaultHeaders(httpReq, req.Referer); err != nil {
			return nil, err
		}
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, api.HandleHTTPError(resp)
	}

	return io.ReadAll(resp.Body)
}

// Get issues a plain GET; non-2xx is reported via statusCode, not err — only a transport failure is an error.
func (c *Client) Get(ctx context.Context, pageURL string) (body []byte, statusCode int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, 0, err
	}
	if err := setDefaultHeaders(req, pageURL); err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func setDefaultHeaders(req *http.Request, referer string) error {
	refURL, err := url.Parse(referer)
	if err != nil {
		return err
	}
	req.Header.Set("Origin", refURL.Scheme+"://"+refURL.Host)
	req.Header.Set("Referer", referer)
	return nil
}
