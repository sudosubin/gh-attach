package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClientDoMultipart_OmitsCookiesScopedToADifferentHost(t *testing.T) {
	var receivedCookie, receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverHost, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	c := NewClient(server.Client(), "test-agent", []*http.Cookie{
		{Name: "user_session", Value: "secret", Domain: serverHost.Hostname(), Path: "/"},
		// Scoped to a different host — must never leak to this request.
		{Name: "other_session", Value: "leak-me-not", Domain: "s3.example.com", Path: "/"},
	})

	if _, err := c.DoMultipart(t.Context(), Request{
		Method: http.MethodPost,
		URL:    server.URL + "/upload",
	}); err != nil {
		t.Fatalf("DoMultipart() error = %v", err)
	}

	if !strings.Contains(receivedCookie, "user_session=secret") {
		t.Fatalf("Cookie = %q, want it to contain the same-host session cookie", receivedCookie)
	}
	if strings.Contains(receivedCookie, "other_session") {
		t.Fatalf("Cookie = %q, leaked a cookie scoped to a different host", receivedCookie)
	}
	if receivedUA != "test-agent" {
		t.Fatalf("User-Agent = %q, want %q (sent regardless of cookie scoping)", receivedUA, "test-agent")
	}
}

func TestClientGet_ScopesCookiesAndReportsStatus(t *testing.T) {
	var receivedCookie, receivedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		receivedUA = r.Header.Get("User-Agent")
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("page body"))
	}))
	defer server.Close()

	serverHost, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	c := NewClient(server.Client(), "test-agent", []*http.Cookie{
		{Name: "user_session", Value: "secret", Domain: serverHost.Hostname(), Path: "/"},
		{Name: "other_session", Value: "leak-me-not", Domain: "s3.example.com", Path: "/"},
	})

	body, status, err := c.Get(t.Context(), server.URL+"/owner/repo/issues/new")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if string(body) != "page body" {
		t.Fatalf("body = %q, want %q", body, "page body")
	}
	if !strings.Contains(receivedCookie, "user_session=secret") {
		t.Fatalf("Cookie = %q, want it to contain the same-host session cookie", receivedCookie)
	}
	if strings.Contains(receivedCookie, "other_session") {
		t.Fatalf("Cookie = %q, leaked a cookie scoped to a different host", receivedCookie)
	}
	// UA is sent regardless of cookie match — a real browser always identifies itself.
	if receivedUA != "test-agent" {
		t.Fatalf("User-Agent = %q, want %q", receivedUA, "test-agent")
	}

	// Non-200 is reported via statusCode, not as an error.
	body, status, err = c.Get(t.Context(), server.URL+"/missing")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	if body != nil {
		t.Fatalf("body = %q, want nil on non-200", body)
	}
}
