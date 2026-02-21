package upload

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestPolicies_UsesResolvedRefererHeader(t *testing.T) {
	const expectedReferer = "https://github.com/owner/repo/commit/abc123"
	var receivedReferer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/policies/assets" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		receivedReferer = r.Header.Get("Referer")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"https://uploads.github.test/upload","form":{},"header":{},"asset":{"id":1}}`))
	}))
	defer server.Close()

	_, err := requestPolicies(
		context.Background(),
		server.Client(),
		server.URL,
		expectedReferer,
		"",
		1,
		"file.txt",
		12,
		"text/plain",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("requestPolicies() error = %v", err)
	}
	if receivedReferer != expectedReferer {
		t.Fatalf("Referer header = %q, want %q", receivedReferer, expectedReferer)
	}
}
