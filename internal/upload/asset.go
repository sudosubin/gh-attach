package upload

import "regexp"

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
