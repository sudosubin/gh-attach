package upload

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sudosubin/gh-attach/internal/ghweb"
)

func TestRequestPolicies_InjectsRefererAndUploadHeaders(t *testing.T) {
	const expectedReferer = "https://github.com/owner/repo/commit/abc123"
	const expectedFetchNonce = "nonce-123"
	const expectedClientVersion = "1.2.3"

	var receivedReferer string
	var receivedRequestedWith string
	var receivedVerifiedFetch string
	var receivedFetchNonce string
	var receivedClientVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/upload/policies/assets" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		receivedReferer = r.Header.Get("Referer")
		receivedRequestedWith = r.Header.Get("X-Requested-With")
		receivedVerifiedFetch = r.Header.Get("GitHub-Verified-Fetch")
		receivedFetchNonce = r.Header.Get("X-Fetch-Nonce")
		receivedClientVersion = r.Header.Get("X-GitHub-Client-Version")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"https://uploads.github.test/upload","form":{},"header":{},"asset":{"id":1}}`))
	}))
	defer server.Close()

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(server.Client(), "", nil),
		isEnterprise: false,
	}
	refererPage := &RefererPage{
		URL:  expectedReferer,
		Body: []byte(`<meta name="fetch-nonce" content="nonce-123"><meta name="release" content="1.2.3">`),
	}

	_, err := u.requestPoliciesCloud(t.Context(), refererPage, "file.txt", 12, "text/plain")
	if err != nil {
		t.Fatalf("requestPoliciesCloud() error = %v", err)
	}
	if receivedReferer != expectedReferer {
		t.Fatalf("Referer header = %q, want %q", receivedReferer, expectedReferer)
	}
	if receivedRequestedWith != "XMLHttpRequest" {
		t.Fatalf("X-Requested-With = %q, want %q", receivedRequestedWith, "XMLHttpRequest")
	}
	if receivedVerifiedFetch != "true" {
		t.Fatalf("GitHub-Verified-Fetch = %q, want %q", receivedVerifiedFetch, "true")
	}
	if receivedFetchNonce != expectedFetchNonce {
		t.Fatalf("X-Fetch-Nonce = %q, want %q", receivedFetchNonce, expectedFetchNonce)
	}
	if receivedClientVersion != expectedClientVersion {
		t.Fatalf("X-GitHub-Client-Version = %q, want %q", receivedClientVersion, expectedClientVersion)
	}
}

func TestRequestPolicies_DoesNotInjectOptionalHeadersWhenMetaMissing(t *testing.T) {
	var receivedFetchNonce string
	var receivedClientVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedFetchNonce = r.Header.Get("X-Fetch-Nonce")
		receivedClientVersion = r.Header.Get("X-GitHub-Client-Version")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"https://uploads.github.test/upload","form":{},"header":{},"asset":{"id":1}}`))
	}))
	defer server.Close()

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(server.Client(), "", nil),
		isEnterprise: false,
	}
	refererPage := &RefererPage{
		URL:  "https://github.com/owner/repo/issues/new",
		Body: []byte{},
	}

	_, err := u.requestPoliciesCloud(t.Context(), refererPage, "file.txt", 12, "text/plain")
	if err != nil {
		t.Fatalf("requestPoliciesCloud() error = %v", err)
	}
	if receivedFetchNonce != "" {
		t.Fatalf("X-Fetch-Nonce = %q, want empty", receivedFetchNonce)
	}
	if receivedClientVersion != "" {
		t.Fatalf("X-GitHub-Client-Version = %q, want empty", receivedClientVersion)
	}
}

func TestRequestPoliciesCloud_IgnoresAuthenticityTokenOnPage(t *testing.T) {
	var receivedAuthToken string
	var receivedFetchNonce string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		receivedAuthToken = r.FormValue("authenticity_token")
		receivedFetchNonce = r.Header.Get("X-Fetch-Nonce")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"https://uploads.github.test/upload","form":{},"header":{},"asset":{"id":1}}`))
	}))
	defer server.Close()

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(server.Client(), "", nil),
		isEnterprise: false,
	}
	refererPage := &RefererPage{
		URL: "https://github.com/owner/repo/issues/new",
		// An unrelated authenticity_token on the page must not be sent on the cloud path.
		Body: []byte(`<input name="authenticity_token" value="incidental-token"><meta name="fetch-nonce" content="nonce-123">`),
	}

	_, err := u.requestPoliciesCloud(t.Context(), refererPage, "file.txt", 12, "text/plain")
	if err != nil {
		t.Fatalf("requestPoliciesCloud() error = %v", err)
	}
	if receivedAuthToken != "" {
		t.Fatalf("authenticity_token = %q, want empty", receivedAuthToken)
	}
	if receivedFetchNonce != "nonce-123" {
		t.Fatalf("X-Fetch-Nonce = %q, want %q", receivedFetchNonce, "nonce-123")
	}
}

func TestRequestPoliciesEnterprise_SendsAuthTokenAndVerifiedFetch(t *testing.T) {
	var receivedAuthToken string
	var receivedVerifiedFetch, receivedFetchNonce, receivedClientVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		receivedAuthToken = r.FormValue("authenticity_token")
		receivedVerifiedFetch = r.Header.Get("GitHub-Verified-Fetch")
		receivedFetchNonce = r.Header.Get("X-Fetch-Nonce")
		receivedClientVersion = r.Header.Get("X-GitHub-Client-Version")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"https://uploads.enterprise.test/upload","form":{},"header":{},"asset":{"id":1}}`))
	}))
	defer server.Close()

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(server.Client(), "", nil),
		isEnterprise: true,
	}
	refererPage := &RefererPage{
		URL: "https://github.enterprise.test/owner/repo/issues/new",
		// Even a stray fetch-nonce meta must not be sent on the enterprise path.
		Body: []byte(`<input name="authenticity_token" value="ghes-token"><meta name="fetch-nonce" content="nonce-123"><meta name="release" content="1.2.3">`),
	}

	_, err := u.requestPoliciesEnterprise(t.Context(), refererPage, "file.txt", 12, "text/plain")
	if err != nil {
		t.Fatalf("requestPoliciesEnterprise() error = %v", err)
	}
	if receivedAuthToken != "ghes-token" {
		t.Fatalf("authenticity_token = %q, want %q", receivedAuthToken, "ghes-token")
	}
	// GitHub-Verified-Fetch is required on both deployment types (verified live); only fetch-nonce is cloud-specific.
	if receivedVerifiedFetch != "true" {
		t.Fatalf("GitHub-Verified-Fetch = %q, want %q", receivedVerifiedFetch, "true")
	}
	if receivedFetchNonce != "" || receivedClientVersion != "" {
		t.Fatalf("fetch-nonce headers = (%q, %q), want both empty", receivedFetchNonce, receivedClientVersion)
	}
}

func TestRequestPoliciesEnterprise_FailsFastWhenAuthTokenMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s: enterprise path should fail before sending one", r.URL.Path)
	}))
	defer server.Close()

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(server.Client(), "", nil),
		isEnterprise: true,
	}
	refererPage := &RefererPage{
		URL:  "https://github.enterprise.test/owner/repo/issues/new",
		Body: []byte{},
	}

	_, err := u.requestPoliciesEnterprise(t.Context(), refererPage, "file.txt", 12, "text/plain")
	if err == nil {
		t.Fatal("requestPoliciesEnterprise() error = nil, want an error")
	}
}

func TestUploadEnterpriseFlow_SkipsFinalize(t *testing.T) {
	tmpFile := t.TempDir() + "/probe.png"
	if err := os.WriteFile(tmpFile, []byte("probe"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	mux := http.NewServeMux()
	// No finalize handler registered — a stray call would 404 and fail Upload().
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/upload/policies/assets", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"` + server.URL + `/media/upload","form":{},"header":{}}`))
	})
	mux.HandleFunc("/media/upload", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"href":"https://github.enterprise.test/user-attachments/assets/abc"}`))
	})

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(server.Client(), "", nil),
		isEnterprise: true,
	}

	refererPage := &RefererPage{
		URL:  server.URL + "/owner/repo/issues/new",
		Body: []byte(`<input name="authenticity_token" value="ghes-token">`),
	}

	asset, err := u.Upload(t.Context(), tmpFile, refererPage)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if asset.Href != "https://github.enterprise.test/user-attachments/assets/abc" {
		t.Fatalf("asset.Href = %q, want the media upload response's href", asset.Href)
	}
}

func TestUploadEnterpriseFlow_MediaHostGetsGHESOriginNotItsOwn(t *testing.T) {
	tmpFile := t.TempDir() + "/probe.png"
	if err := os.WriteFile(tmpFile, []byte("probe"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var mediaReferer, mediaOrigin string

	ghesMux := http.NewServeMux()
	ghesServer := httptest.NewServer(ghesMux)
	defer ghesServer.Close()

	mediaMux := http.NewServeMux()
	mediaServer := httptest.NewServer(mediaMux)
	defer mediaServer.Close()

	ghesMux.HandleFunc("/upload/policies/assets", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"` + mediaServer.URL + `/media/upload","form":{},"header":{}}`))
	})
	mediaMux.HandleFunc("/media/upload", func(w http.ResponseWriter, r *http.Request) {
		mediaReferer = r.Header.Get("Referer")
		mediaOrigin = r.Header.Get("Origin")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"href":"https://github.enterprise.test/user-attachments/assets/abc"}`))
	})

	u := &Uploader{
		baseURL:      ghesServer.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(ghesServer.Client(), "", nil),
		isEnterprise: true,
	}

	refererPage := &RefererPage{
		URL:  ghesServer.URL + "/owner/repo/issues/new",
		Body: []byte(`<input name="authenticity_token" value="ghes-token">`),
	}

	if _, err := u.Upload(t.Context(), tmpFile, refererPage); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	// media.<host> is a different host, so referer/origin are the GHES host's bare origin, not the media host's.
	want := ghesServer.URL + "/"
	if mediaReferer != want {
		t.Fatalf("media upload Referer = %q, want %q", mediaReferer, want)
	}
	if ghesOrigin := mustOrigin(t, ghesServer.URL); mediaOrigin != ghesOrigin {
		t.Fatalf("media upload Origin = %q, want %q (the GHES host, not the media host)", mediaOrigin, ghesOrigin)
	}
}

func TestUploadCloudFlow_UsesRefererPageURLConsistently(t *testing.T) {
	tmpFile := t.TempDir() + "/probe.png"
	if err := os.WriteFile(tmpFile, []byte("probe"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var policiesReferer, uploadReferer, uploadOrigin, finalizeReferer string
	var finalizeFetchNonce, finalizeClientVersion string

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	s3Mux := http.NewServeMux()
	s3Server := httptest.NewServer(s3Mux)
	defer s3Server.Close()

	const refererURL = "issues/new-referer" // sentinel; real value set below once server.URL is known

	mux.HandleFunc("/upload/policies/assets", func(w http.ResponseWriter, r *http.Request) {
		policiesReferer = r.Header.Get("Referer")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"upload_url":"` + s3Server.URL + `/s3","form":{},"header":{},"asset_upload_url":"/upload/assets/1","asset_upload_authenticity_token":"final-token"}`))
	})
	s3Mux.HandleFunc("/s3", func(w http.ResponseWriter, r *http.Request) {
		uploadReferer = r.Header.Get("Referer")
		uploadOrigin = r.Header.Get("Origin")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/upload/assets/1", func(w http.ResponseWriter, r *http.Request) {
		finalizeReferer = r.Header.Get("Referer")
		finalizeFetchNonce = r.Header.Get("X-Fetch-Nonce")
		finalizeClientVersion = r.Header.Get("X-GitHub-Client-Version")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"href":"https://github.test/user-attachments/assets/final"}`))
	})

	u := &Uploader{
		baseURL:      server.URL,
		repositoryID: 1,
		client:       ghweb.NewClient(server.Client(), "", nil),
		isEnterprise: false,
	}

	refererPage := &RefererPage{
		URL:  server.URL + "/" + refererURL,
		Body: []byte(`<meta name="fetch-nonce" content="nonce-xyz"><meta name="release" content="v1">`),
	}

	asset, err := u.Upload(t.Context(), tmpFile, refererPage)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if asset.Href != "https://github.test/user-attachments/assets/final" {
		t.Fatalf("asset.Href = %q", asset.Href)
	}

	for label, got := range map[string]string{
		"policies Referer": policiesReferer,
		"upload Referer":   uploadReferer,
		"finalize Referer": finalizeReferer,
	} {
		if got != refererPage.URL {
			t.Fatalf("%s = %q, want %q (all three steps should use the same referer page URL)", label, got, refererPage.URL)
		}
	}
	// S3 is a different host, so Origin must reflect the initiating host, not S3's.
	if serverOrigin := mustOrigin(t, server.URL); uploadOrigin != serverOrigin {
		t.Fatalf("upload Origin = %q, want %q (the initiating host, not S3's)", uploadOrigin, serverOrigin)
	}
	if finalizeFetchNonce != "nonce-xyz" {
		t.Fatalf("finalize X-Fetch-Nonce = %q, want %q", finalizeFetchNonce, "nonce-xyz")
	}
	if finalizeClientVersion != "v1" {
		t.Fatalf("finalize X-GitHub-Client-Version = %q, want %q", finalizeClientVersion, "v1")
	}
}
