package upload

import "encoding/json"

type stringMap map[string]string

func (m *stringMap) UnmarshalJSON(data []byte) error {
	raw := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = make(map[string]string, len(raw))
	for k, v := range raw {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			(*m)[k] = s
		} else {
			(*m)[k] = string(v)
		}
	}
	return nil
}

type Asset struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	ContentType  string  `json:"content_type"`
	Href         string  `json:"href"`
	OriginalName *string `json:"original_name"`
}

type policiesResponse struct {
	UploadURL                    string    `json:"upload_url"`
	UploadAuthenticityToken      string    `json:"upload_authenticity_token"`
	Form                         stringMap `json:"form"`
	Header                       stringMap `json:"header"`
	Asset                        Asset     `json:"asset"`
	AssetUploadURL               string    `json:"asset_upload_url"`
	AssetUploadAuthenticityToken string    `json:"asset_upload_authenticity_token"`
	SameOrigin                   bool      `json:"same_origin"`
}
