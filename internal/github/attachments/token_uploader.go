package attachments

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"
)

// TokenUploader sends media to GitHub Cloud using gh API authentication.
type TokenUploader struct {
	host         string
	repositoryID int64
	client       *http.Client
}

func NewTokenUploader(host, token string, repositoryID int64, client *http.Client) (*TokenUploader, error) {
	if auth.IsEnterprise(host) {
		return nil, errors.New("token uploads are not supported on GitHub Enterprise Server")
	}
	if token == "" || repositoryID <= 0 {
		return nil, errors.New("token upload requires authentication and a repository id")
	}
	options := api.ClientOptions{Host: host, AuthToken: token, CheckRedirect: tokenUploadRedirect}
	if client != nil {
		options.Transport = client.Transport
		options.Timeout = client.Timeout
	}
	httpClient, err := api.NewHTTPClient(options)
	if err != nil {
		return nil, err
	}
	return &TokenUploader{host: host, repositoryID: repositoryID, client: httpClient}, nil
}

func SupportsTokenUpload(filePath string) bool {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".mp4", ".mov", ".webm":
		return true
	default:
		return false
	}
}

func (u *TokenUploader) Upload(ctx context.Context, filePath string) (Asset, error) {
	info, err := StatUploadFile(filePath)
	if err != nil {
		return Asset{}, err
	}
	if !SupportsTokenUpload(filePath) {
		return Asset{}, fmt.Errorf("%s is not supported by the token upload API", filePath)
	}
	if info.Size() == 0 {
		return Asset{}, fmt.Errorf("%s is empty", filePath)
	}
	name := filepath.Base(filePath)
	contentType := contentTypeForFile(name)
	maxSize := int64(10 * 1024 * 1024)
	if strings.HasPrefix(contentType, "video/") {
		maxSize = 100 * 1024 * 1024
	}
	if info.Size() > maxSize {
		return Asset{}, fmt.Errorf("%s exceeds the token upload limit of %d MiB", filePath, maxSize/(1024*1024))
	}

	uploadURL := url.URL{Scheme: "https", Host: "uploads." + u.host, Path: "/user-attachments/assets"}
	uploadURL.RawQuery = url.Values{
		"name":          {name},
		"content_type":  {contentType},
		"repository_id": {strconv.FormatInt(u.repositoryID, 10)},
	}.Encode()
	open := func() (io.ReadCloser, error) { return os.Open(filePath) }
	file, err := open()
	if err != nil {
		return Asset{}, fmt.Errorf("open upload file: %w", err)
	}
	defer func() { _ = file.Close() }()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL.String(), file)
	if err != nil {
		return Asset{}, err
	}
	req.ContentLength = info.Size()
	req.GetBody = open
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := u.client.Do(req)
	if err != nil {
		return Asset{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Asset{}, api.HandleHTTPError(resp)
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.UnmarshalRead(resp.Body, &result); err != nil {
		return Asset{}, fmt.Errorf("decode token upload response: %w", err)
	}
	assetURL, err := ParseURL(result.URL)
	if err != nil || !strings.EqualFold(assetURL.Host, u.host) ||
		!strings.HasPrefix(assetURL.EscapedPath(), "/user-attachments/assets/") {
		return Asset{}, errors.New("token upload returned an invalid asset URL")
	}
	return Asset{Name: name, Size: info.Size(), ContentType: contentType, Href: assetURL.String()}, nil
}

func tokenUploadRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("token upload stopped after 10 redirects")
	}
	if req.URL.Scheme != "https" || !strings.EqualFold(req.URL.Host, via[0].URL.Host) || req.Method != http.MethodPost {
		return errors.New("token upload refused a redirect changing its host, scheme, or method")
	}
	return nil
}
