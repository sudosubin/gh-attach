package attachments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/github/web"
)

// Uploader carries the shared session context for GitHub user-attachment uploads.
type Uploader struct {
	baseURL      string // e.g. "https://github.com"
	client       *web.Client
	isEnterprise bool
}

// contentTypesByExtension contains known browser-compatible MIME hints.
// Empty content types intentionally let the server infer from the extension.
var contentTypesByExtension = map[string]string{
	".bmp":        "image/bmp",
	".c":          "",
	".copilotmd":  "",
	".cpp":        "",
	".cpuprofile": "",
	".cs":         "",
	".css":        "text/css",
	".csv":        "text/csv",
	".debug":      "",
	".dmp":        "",
	".doc":        "application/msword",
	".docx":       "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".drawio":     "",
	".eml":        "message/rfc822",
	".fodg":       "application/vnd.oasis.opendocument.graphics",
	".fodp":       "application/vnd.oasis.opendocument.presentation",
	".fods":       "application/vnd.oasis.opendocument.spreadsheet",
	".fodt":       "application/vnd.oasis.opendocument.text",
	".gif":        "image/gif",
	".gz":         "application/x-gzip",
	".htm":        "text/html",
	".html":       "text/html",
	".ipynb":      "",
	".java":       "",
	".jpeg":       "image/jpeg",
	".jpg":        "image/jpeg",
	".js":         "text/javascript",
	".json":       "application/json",
	".jsonc":      "",
	".log":        "",
	".md":         "text/markdown",
	".mov":        "video/quicktime",
	".mp3":        "audio/mpeg",
	".mp4":        "video/mp4",
	".msg":        "",
	".odf":        "application/vnd.oasis.opendocument.formula",
	".odg":        "application/vnd.oasis.opendocument.graphics",
	".odp":        "application/vnd.oasis.opendocument.presentation",
	".ods":        "application/vnd.oasis.opendocument.spreadsheet",
	".odt":        "application/vnd.oasis.opendocument.text",
	".patch":      "",
	".pdb":        "application/octet-stream",
	".pdf":        "application/pdf",
	".php":        "text/php",
	".png":        "image/png",
	".pptx":       "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".py":         "text/x-python-script",
	".rtf":        "text/rtf",
	".sh":         "text/x-sh",
	".sql":        "",
	".svg":        "image/svg+xml",
	".tgz":        "application/gzip",
	".tif":        "image/tiff",
	".tiff":       "image/tiff",
	".ts":         "",
	".tsv":        "text/tab-separated-values",
	".tsx":        "",
	".txt":        "text/plain",
	".wav":        "audio/wav",
	".webm":       "video/webm",
	".webp":       "image/webp",
	".xls":        "application/vnd.ms-excel",
	".xlsm":       "application/vnd.ms-excel.sheet.macroenabled.12",
	".xlsx":       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xml":        "text/xml",
	".yaml":       "application/x-yaml",
	".yml":        "application/x-yaml",
	".zip":        "application/zip",
}

func NewUploader(host string, session browserprovider.BrowserSession, client *http.Client) (*Uploader, error) {
	return &Uploader{
		baseURL:      "https://" + host,
		client:       web.NewClient(client, session.UserAgent, session.Cookies),
		isEnterprise: auth.IsEnterprise(host),
	}, nil
}

func (u *Uploader) ResolveRefererPage(ctx context.Context, fetchers []RefererPageFetcher) (*RefererPage, error) {
	if len(fetchers) == 0 {
		return nil, fmt.Errorf("no referer page fetchers configured")
	}

	var errs []error
	for _, fetcher := range fetchers {
		page, err := fetcher.Fetch(ctx, u.client)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if page != nil {
			return page, nil
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("failed to resolve accessible referer page")
}

func (u *Uploader) Upload(ctx context.Context, filePath string, refererPage *RefererPage, repositoryID int64) (Asset, error) {
	if refererPage == nil || strings.TrimSpace(refererPage.URL) == "" {
		return Asset{}, fmt.Errorf("invalid referer page")
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return Asset{}, fmt.Errorf("file: %w", err)
	}
	if fileInfo.IsDir() {
		return Asset{}, fmt.Errorf("%s is a directory", filePath)
	}

	fileName := filepath.Base(filePath)
	contentType := contentTypeForFile(fileName)

	if u.isEnterprise {
		return u.uploadEnterpriseFlow(ctx, filePath, refererPage, repositoryID, fileName, fileInfo.Size(), contentType)
	}
	return u.uploadCloudFlow(ctx, filePath, refererPage, repositoryID, fileName, fileInfo.Size(), contentType)
}

func contentTypeForFile(fileName string) string {
	extension := strings.ToLower(filepath.Ext(fileName))
	return contentTypesByExtension[extension]
}

// uploadCloudFlow is github.com's 3-step upload: policies -> S3 -> finalize.
func (u *Uploader) uploadCloudFlow(ctx context.Context, filePath string, refererPage *RefererPage, repositoryID int64, fileName string, fileSize int64, contentType string) (Asset, error) {
	policies, err := u.requestPoliciesCloud(ctx, refererPage, repositoryID, fileName, fileSize, contentType)
	if err != nil {
		return Asset{}, err
	}

	uploadedAsset, err := u.uploadBinaryCloud(ctx, filePath, contentType, policies, refererPage.URL)
	if err != nil {
		return Asset{}, err
	}

	asset, err := u.finalizeAssetCloud(ctx, policies, refererPage)
	if err != nil {
		if uploadedAsset.Href != "" {
			return uploadedAsset, nil
		}
		if policies.Asset.Href != "" {
			return policies.Asset, nil
		}
		return Asset{}, err
	}

	if asset.Href == "" && uploadedAsset.Href != "" {
		return uploadedAsset, nil
	}

	return asset, nil
}

// uploadEnterpriseFlow is GHES's 2-step upload: policies -> media host (no finalize step).
func (u *Uploader) uploadEnterpriseFlow(ctx context.Context, filePath string, refererPage *RefererPage, repositoryID int64, fileName string, fileSize int64, contentType string) (Asset, error) {
	policies, err := u.requestPoliciesEnterprise(ctx, refererPage, repositoryID, fileName, fileSize, contentType)
	if err != nil {
		return Asset{}, err
	}
	return u.uploadBinaryEnterprise(ctx, filePath, contentType, policies, u.baseURL+"/")
}

// requestPoliciesCloud authenticates via X-Fetch-Nonce/X-GitHub-Client-Version, never authenticity_token.
func (u *Uploader) requestPoliciesCloud(ctx context.Context, refererPage *RefererPage, repositoryID int64, fileName string, fileSize int64, contentType string) (policiesResponse, error) {
	refererMeta := refererPage.Meta

	headers := map[string]string{}
	if strings.TrimSpace(refererMeta.FetchNonce) != "" {
		headers["X-Fetch-Nonce"] = refererMeta.FetchNonce
	}
	if strings.TrimSpace(refererMeta.GitHubClientVersion) != "" {
		headers["X-GitHub-Client-Version"] = refererMeta.GitHubClientVersion
	}

	return u.requestPolicies(ctx, refererPage.URL, policiesFields(repositoryID, fileName, fileSize, contentType), headers)
}

// requestPoliciesEnterprise authenticates via authenticity_token and fails fast if it's missing from the referer page.
func (u *Uploader) requestPoliciesEnterprise(ctx context.Context, refererPage *RefererPage, repositoryID int64, fileName string, fileSize int64, contentType string) (policiesResponse, error) {
	refererMeta := refererPage.Meta
	if refererMeta.AuthenticityToken == "" {
		return policiesResponse{}, fmt.Errorf("enterprise host requires an authenticity_token, but none was found on the referer page")
	}

	fields := policiesFields(repositoryID, fileName, fileSize, contentType)
	fields["authenticity_token"] = refererMeta.AuthenticityToken

	return u.requestPolicies(ctx, refererPage.URL, fields, map[string]string{})
}

func policiesFields(repositoryID int64, fileName string, fileSize int64, contentType string) map[string]string {
	return map[string]string{
		"repository_id": strconv.FormatInt(repositoryID, 10),
		"name":          fileName,
		"size":          strconv.FormatInt(fileSize, 10),
		"content_type":  contentType,
	}
}

// requestPolicies posts to /upload/policies/assets; GitHub-Verified-Fetch is required on both deployment types.
func (u *Uploader) requestPolicies(ctx context.Context, refererURL string, fields, headers map[string]string) (policiesResponse, error) {
	headers["Accept"] = "application/json"
	headers["X-Requested-With"] = "XMLHttpRequest"
	headers["GitHub-Verified-Fetch"] = "true"

	body, err := u.client.DoMultipart(ctx, web.Request{
		Method:  http.MethodPost,
		URL:     u.baseURL + "/upload/policies/assets",
		Fields:  fields,
		Headers: headers,
		Referer: refererURL,
	})
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

// uploadBinaryCloud PUTs to the S3 presigned URL; an empty/unparsable body isn't an error since finalizeAssetCloud fills in the asset.
func (u *Uploader) uploadBinaryCloud(ctx context.Context, filePath string, contentType string, policies policiesResponse, refererURL string) (Asset, error) {
	headers := make(map[string]string, len(policies.Header)+1)
	if contentType != "" {
		headers["X-File-Content-Type"] = contentType
	}
	maps.Copy(headers, policies.Header)

	body, err := u.client.DoMultipart(ctx, web.Request{
		Method:   http.MethodPost,
		URL:      policies.UploadURL,
		Fields:   policies.Form,
		Headers:  headers,
		FilePath: filePath,
		Referer:  refererURL,
	})
	if err != nil {
		return Asset{}, err
	}
	if len(body) == 0 {
		return Asset{}, nil
	}

	var asset Asset
	if err := json.Unmarshal(body, &asset); err != nil {
		return Asset{}, nil
	}
	return asset, nil
}

// uploadBinaryEnterprise POSTs to the media host; the response IS the finished asset, so a missing href is a real error.
func (u *Uploader) uploadBinaryEnterprise(ctx context.Context, filePath string, contentType string, policies policiesResponse, refererURL string) (Asset, error) {
	fields := make(map[string]string, len(policies.Form)+1)
	maps.Copy(fields, policies.Form)
	if policies.UploadAuthenticityToken != "" {
		fields["authenticity_token"] = policies.UploadAuthenticityToken
	}

	headers := make(map[string]string, len(policies.Header)+1)
	if contentType != "" {
		headers["X-File-Content-Type"] = contentType
	}
	maps.Copy(headers, policies.Header)

	body, err := u.client.DoMultipart(ctx, web.Request{
		Method:   http.MethodPost,
		URL:      policies.UploadURL,
		Fields:   fields,
		Headers:  headers,
		FilePath: filePath,
		Referer:  refererURL,
	})
	if err != nil {
		return Asset{}, err
	}

	var asset Asset
	if err := json.Unmarshal(body, &asset); err != nil {
		return Asset{}, fmt.Errorf("enterprise media upload response: %w", err)
	}
	if asset.Href == "" {
		return Asset{}, fmt.Errorf("enterprise media upload response missing href")
	}
	return asset, nil
}

// finalizeAssetCloud PUTs to asset_upload_url to mark the asset ready; only the cloud flow calls this.
func (u *Uploader) finalizeAssetCloud(ctx context.Context, policies policiesResponse, refererPage *RefererPage) (Asset, error) {
	if policies.AssetUploadURL == "" {
		return policies.Asset, nil
	}

	finalURL := policies.AssetUploadURL
	if strings.HasPrefix(finalURL, "/") {
		finalURL = u.baseURL + finalURL
	}

	// Same fetch-nonce pair as the policies request, from the same referer page.
	refererMeta := refererPage.Meta
	headers := map[string]string{
		"Accept":           "application/json",
		"X-Requested-With": "XMLHttpRequest",
	}
	if strings.TrimSpace(refererMeta.FetchNonce) != "" {
		headers["X-Fetch-Nonce"] = refererMeta.FetchNonce
	}
	if strings.TrimSpace(refererMeta.GitHubClientVersion) != "" {
		headers["X-GitHub-Client-Version"] = refererMeta.GitHubClientVersion
	}

	body, err := u.client.DoMultipart(ctx, web.Request{
		Method:  http.MethodPut,
		URL:     finalURL,
		Fields:  map[string]string{"authenticity_token": policies.AssetUploadAuthenticityToken},
		Headers: headers,
		Referer: refererPage.URL,
	})
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
