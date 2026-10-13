package attachments

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/sudosubin/gh-attach/internal/github/web"
)

// Uploader carries the shared session context for GitHub user-attachment uploads.
type Uploader struct {
	baseURL      string // e.g. "https://github.com"
	client       *web.Client
	isEnterprise bool
}

// uploadFile is one file's upload context, built once by Upload and passed through every step.
type uploadFile struct {
	path         string
	name         string
	size         int64
	contentType  string
	repositoryID int64
	refererPage  *RefererPage
}

var errMissingHref = errors.New("upload response missing href")

func NewUploader(host string, session web.Session, client *http.Client) (*Uploader, error) {
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

	fileInfo, err := StatUploadFile(filePath)
	if err != nil {
		return Asset{}, err
	}

	fileName := filepath.Base(filePath)
	f := uploadFile{
		path:         filePath,
		name:         fileName,
		size:         fileInfo.Size(),
		contentType:  contentTypeForFile(fileName),
		repositoryID: repositoryID,
		refererPage:  refererPage,
	}

	var asset Asset
	switch {
	case !u.isEnterprise:
		asset, err = u.uploadCloudFlow(ctx, f)
	case refererPage.Meta.UploadForm == uploadFormFileAttachment:
		asset, err = u.uploadEnterpriseFileAttachmentFlow(ctx, f)
	case refererPage.Meta.UploadForm == uploadFormCommentBox:
		asset, err = u.uploadEnterpriseCommentBoxFlow(ctx, f)
	default:
		err = fmt.Errorf("unrecognized upload form on %s", refererPage.URL)
	}
	if err != nil {
		return Asset{}, err
	}
	asset.Name = cmp.Or(asset.Name, f.name)
	asset.ContentType = cmp.Or(asset.ContentType, f.contentType)
	return asset, nil
}

// uploadCloudFlow is github.com's upload: policy -> storage (S3) -> completion.
func (u *Uploader) uploadCloudFlow(ctx context.Context, f uploadFile) (Asset, error) {
	policy, err := u.requestPolicyCommentBox(ctx, f)
	if err != nil {
		return Asset{}, err
	}

	storedAsset, err := u.requestStorage(ctx, f, policy, f.refererPage.URL)
	if err != nil {
		return Asset{}, err
	}

	asset, err := u.requestCompletion(ctx, f, policy)
	if err != nil {
		if storedAsset.Href != "" {
			return storedAsset, nil
		}
		if policy.Asset.Href != "" {
			return policy.Asset, nil
		}
		return Asset{}, err
	}

	if asset.Href == "" && storedAsset.Href != "" {
		return storedAsset, nil
	}

	return asset, nil
}

// uploadEnterpriseFileAttachmentFlow is GHES's <file-attachment> upload: policy -> storage (no completion).
func (u *Uploader) uploadEnterpriseFileAttachmentFlow(ctx context.Context, f uploadFile) (Asset, error) {
	policy, err := u.requestPolicyFileAttachment(ctx, f)
	if err != nil {
		return Asset{}, err
	}

	storedAsset, err := u.requestStorage(ctx, f, policy, u.baseURL+"/")
	if err != nil {
		return Asset{}, err
	}
	if storedAsset.Href == "" {
		return Asset{}, errMissingHref
	}
	return storedAsset, nil
}

// uploadEnterpriseCommentBoxFlow is GHES's CommentBox upload: policy -> storage -> completion if requested.
func (u *Uploader) uploadEnterpriseCommentBoxFlow(ctx context.Context, f uploadFile) (Asset, error) {
	policy, err := u.requestPolicyCommentBox(ctx, f)
	if err != nil {
		return Asset{}, err
	}

	storedAsset, err := u.requestStorage(ctx, f, policy, u.baseURL+"/")
	if err != nil {
		return Asset{}, err
	}
	if policy.AssetUploadURL == "" {
		if storedAsset.Href == "" {
			return Asset{}, errMissingHref
		}
		return storedAsset, nil
	}

	asset, err := u.requestCompletion(ctx, f, policy)
	if err == nil && asset.Href != "" {
		return asset, nil
	}
	if storedAsset.Href != "" {
		return storedAsset, nil // a failed completion isn't fatal once storage returned a usable href
	}
	return Asset{}, cmp.Or(err, errMissingHref)
}

// requestPolicyCommentBox authenticates via X-Fetch-Nonce/X-GitHub-Client-Version, never authenticity_token.
func (u *Uploader) requestPolicyCommentBox(ctx context.Context, f uploadFile) (policyResponse, error) {
	return u.requestPolicy(ctx, f, policyFields(f), fetchNonceHeaders(f.refererPage.Meta))
}

// requestPolicyFileAttachment authenticates via authenticity_token and fails fast if it's missing from the referer page.
func (u *Uploader) requestPolicyFileAttachment(ctx context.Context, f uploadFile) (policyResponse, error) {
	token := f.refererPage.Meta.AuthenticityToken
	if token == "" {
		return policyResponse{}, fmt.Errorf("enterprise host requires an authenticity_token, but none was found on the referer page")
	}

	fields := policyFields(f)
	fields["authenticity_token"] = token

	return u.requestPolicy(ctx, f, fields, map[string]string{})
}

func policyFields(f uploadFile) map[string]string {
	return map[string]string{
		"repository_id": strconv.FormatInt(f.repositoryID, 10),
		"name":          f.name,
		"size":          strconv.FormatInt(f.size, 10),
		"content_type":  f.contentType,
	}
}

// fetchNonceHeaders is the CommentBox CSRF context shared by its policy and completion requests.
func fetchNonceHeaders(meta refererPageMetadata) map[string]string {
	headers := map[string]string{}
	if strings.TrimSpace(meta.FetchNonce) != "" {
		headers["X-Fetch-Nonce"] = meta.FetchNonce
	}
	if strings.TrimSpace(meta.GitHubClientVersion) != "" {
		headers["X-GitHub-Client-Version"] = meta.GitHubClientVersion
	}
	return headers
}

// requestPolicy posts to /upload/policies/assets; GitHub-Verified-Fetch is required on both deployment types.
func (u *Uploader) requestPolicy(ctx context.Context, f uploadFile, fields, headers map[string]string) (policyResponse, error) {
	headers["Accept"] = "application/json"
	headers["X-Requested-With"] = "XMLHttpRequest"
	headers["GitHub-Verified-Fetch"] = "true"

	body, err := u.client.DoMultipart(ctx, web.Request{
		Method:  http.MethodPost,
		URL:     u.baseURL + "/upload/policies/assets",
		Fields:  fields,
		Headers: headers,
		Referer: f.refererPage.URL,
	})
	if err != nil {
		return policyResponse{}, err
	}

	var out policyResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return policyResponse{}, err
	}
	if out.UploadURL == "" {
		return policyResponse{}, fmt.Errorf("policy response missing upload_url")
	}
	return out, nil
}

// requestStorage POSTs the file to upload_url; same-origin storage (GHES media host) also takes upload_authenticity_token.
func (u *Uploader) requestStorage(ctx context.Context, f uploadFile, policy policyResponse, refererURL string) (Asset, error) {
	fields := make(map[string]string, len(policy.Form)+1)
	maps.Copy(fields, policy.Form)
	if policy.SameOrigin && policy.UploadAuthenticityToken != "" {
		fields["authenticity_token"] = policy.UploadAuthenticityToken
	}

	headers := make(map[string]string, len(policy.Header)+1)
	if f.contentType != "" {
		headers["X-File-Content-Type"] = f.contentType
	}
	maps.Copy(headers, policy.Header)

	body, err := u.client.DoMultipart(ctx, web.Request{
		Method:   http.MethodPost,
		URL:      policy.UploadURL,
		Fields:   fields,
		Headers:  headers,
		FilePath: f.path,
		Referer:  refererURL,
	})
	if err != nil {
		return Asset{}, err
	}

	// S3 answers without a JSON asset; flows decide whether they need one.
	var asset Asset
	if json.Unmarshal(body, &asset) != nil {
		return Asset{}, nil
	}
	return asset, nil
}

// requestCompletion PUTs to asset_upload_url to mark the asset ready.
func (u *Uploader) requestCompletion(ctx context.Context, f uploadFile, policy policyResponse) (Asset, error) {
	if policy.AssetUploadURL == "" {
		return policy.Asset, nil
	}

	completionURL := policy.AssetUploadURL
	if strings.HasPrefix(completionURL, "/") {
		completionURL = u.baseURL + completionURL
	}

	// Same fetch-nonce pair as the policy request, from the same referer page.
	headers := fetchNonceHeaders(f.refererPage.Meta)
	headers["Accept"] = "application/json"
	headers["X-Requested-With"] = "XMLHttpRequest"

	body, err := u.client.DoMultipart(ctx, web.Request{
		Method:  http.MethodPut,
		URL:     completionURL,
		Fields:  map[string]string{"authenticity_token": policy.AssetUploadAuthenticityToken},
		Headers: headers,
		Referer: f.refererPage.URL,
	})
	if err != nil {
		return Asset{}, err
	}

	var asset Asset
	if err := json.Unmarshal(body, &asset); err != nil {
		return policy.Asset, nil
	}
	if asset.Href == "" && policy.Asset.Href != "" {
		return policy.Asset, nil
	}
	return asset, nil
}
