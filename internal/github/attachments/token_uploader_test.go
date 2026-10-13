package attachments

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type tokenUploadTestTransport func(*http.Request) (*http.Response, error)

func (f tokenUploadTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestTokenUploader_UploadRedirects(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		location string
		wantErr  bool
	}{
		{name: "same host replays body", status: http.StatusTemporaryRedirect, location: "https://uploads.github.com/retry"},
		{name: "other host blocked", status: http.StatusTemporaryRedirect, location: "https://example.com/upload", wantErr: true},
		{name: "HTTP downgrade blocked", status: http.StatusTemporaryRedirect, location: "http://uploads.github.com/retry", wantErr: true},
		{name: "method change blocked", status: http.StatusFound, location: "https://uploads.github.com/retry", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: tokenUploadTestTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				data, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				if string(data) != "probe" || req.ContentLength != int64(len(data)) || req.Method != http.MethodPost {
					t.Errorf("request %d method/body/length = %s, %q, %d", requests, req.Method, data, req.ContentLength)
				}
				if req.Header.Get("Authorization") != "token test-token" {
					t.Errorf("Authorization = %q, want token test-token", req.Header.Get("Authorization"))
				}
				if requests == 1 && (req.URL.Query().Get("name") != "image.png" || req.URL.Query().Get("content_type") != "image/png") {
					t.Errorf("upload query = %v", req.URL.Query())
				}
				status, body := http.StatusCreated, `{"url":"https://github.com/user-attachments/assets/id"}`
				headers := make(http.Header)
				if requests == 1 {
					status, body = tt.status, ""
					headers.Set("Location", tt.location)
				}
				return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
			})}
			uploader, err := NewTokenUploader("github.com", "test-token", 123, client)
			if err != nil {
				t.Fatalf("NewTokenUploader() error = %v", err)
			}
			asset, err := uploader.Upload(t.Context(), writeProbeFile(t, "image.png"))
			if tt.wantErr {
				if err == nil || requests != 1 {
					t.Fatalf("Upload() error = %v, requests = %d; want rejection before following redirect", err, requests)
				}
			} else if err != nil || requests != 2 || asset.Href != "https://github.com/user-attachments/assets/id" {
				t.Fatalf("Upload() = %v, %v, requests = %d; want successful replay", asset, err, requests)
			}
		})
	}
}
