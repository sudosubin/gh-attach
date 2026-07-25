package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func mustRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	return req
}

func TestCheckRedirect_StripsCookieOnCrossSubdomainRedirect(t *testing.T) {
	c := NewClient(nil, "test-agent", []*http.Cookie{
		{Name: "user_session", Value: "secret", Domain: "github.test", Path: "/"},
	})

	initial := mustRequest(t, "https://github.test/issues/new")
	next := mustRequest(t, "https://media.github.test/user/1/files")
	next.Header.Set("Cookie", "user_session=secret")

	if err := c.checkRedirect(next, []*http.Request{initial}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if next.Header.Get("Cookie") != "" {
		t.Fatalf("Cookie = %q, want empty", next.Header.Get("Cookie"))
	}
}

func TestCheckRedirect_StripsAuthHeadersOnCrossOriginRedirect(t *testing.T) {
	c := NewClient(nil, "test-agent", nil)

	initial := mustRequest(t, "https://github.test/upload/policies/assets")
	next := mustRequest(t, "https://media.github.test/user/1/files")
	next.Header.Set("Authorization", "RemoteAuth token")
	next.Header.Set("GitHub-Remote-Auth", "token")

	if err := c.checkRedirect(next, []*http.Request{initial}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if next.Header.Get("Authorization") != "" {
		t.Fatalf("Authorization = %q, want empty", next.Header.Get("Authorization"))
	}
	if next.Header.Get("GitHub-Remote-Auth") != "" {
		t.Fatalf("GitHub-Remote-Auth = %q, want empty", next.Header.Get("GitHub-Remote-Auth"))
	}
}

func TestCheckRedirect_PreservesAuthHeadersOnSameOriginRedirect(t *testing.T) {
	c := NewClient(nil, "test-agent", nil)

	initial := mustRequest(t, "https://media.github.test/a")
	next := mustRequest(t, "https://media.github.test/b")
	next.Header.Set("Authorization", "RemoteAuth token")
	next.Header.Set("GitHub-Remote-Auth", "token")

	if err := c.checkRedirect(next, []*http.Request{initial}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if next.Header.Get("Authorization") != "RemoteAuth token" {
		t.Fatalf("Authorization = %q, want preserved", next.Header.Get("Authorization"))
	}
	if next.Header.Get("GitHub-Remote-Auth") != "token" {
		t.Fatalf("GitHub-Remote-Auth = %q, want preserved", next.Header.Get("GitHub-Remote-Auth"))
	}
}

func TestCheckRedirect_KeepsAuthHeadersStrippedAfterReturningToOriginalOrigin(t *testing.T) {
	c := NewClient(nil, "test-agent", nil)

	initial := mustRequest(t, "https://github.test/a")
	crossOrigin := mustRequest(t, "https://media.github.test/b")
	back := mustRequest(t, "https://github.test/c")
	back.Header.Set("Authorization", "RemoteAuth token")

	if err := c.checkRedirect(back, []*http.Request{initial, crossOrigin}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if back.Header.Get("Authorization") != "" {
		t.Fatalf("Authorization = %q, want empty (stays stripped once origin was left)", back.Header.Get("Authorization"))
	}
}

func TestCheckRedirect_RecomputesCookieForSameHost(t *testing.T) {
	c := NewClient(nil, "test-agent", []*http.Cookie{
		{Name: "user_session", Value: "secret", Domain: "github.test", Path: "/"},
	})

	initial := mustRequest(t, "https://github.test/a")
	next := mustRequest(t, "https://github.test/b")

	if err := c.checkRedirect(next, []*http.Request{initial}); err != nil {
		t.Fatalf("checkRedirect() error = %v", err)
	}
	if next.Header.Get("Cookie") != "user_session=secret" {
		t.Fatalf("Cookie = %q, want %q", next.Header.Get("Cookie"), "user_session=secret")
	}
}

func TestCheckRedirect_RejectsHTTPSDowngrade(t *testing.T) {
	c := NewClient(nil, "test-agent", nil)

	initial := mustRequest(t, "https://github.test/a")
	next := mustRequest(t, "http://github.test/b")

	if err := c.checkRedirect(next, []*http.Request{initial}); err == nil {
		t.Fatal("checkRedirect() error = nil, want an error")
	}
}

func TestCheckRedirect_StopsAfterMaxRedirects(t *testing.T) {
	c := NewClient(nil, "test-agent", nil)

	initial := mustRequest(t, "https://github.test/a")
	next := mustRequest(t, "https://github.test/b")

	via := make([]*http.Request, maxRedirects)
	for i := range via {
		via[i] = initial
	}

	if err := c.checkRedirect(next, via); err == nil {
		t.Fatal("checkRedirect() error = nil, want an error")
	}
}

func TestClientGet_FollowsSameHostRedirect(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	var receivedCookie string
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/b", http.StatusFound)
	})
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	serverHost, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	c := NewClient(server.Client(), "test-agent", []*http.Cookie{
		{Name: "user_session", Value: "secret", Domain: serverHost.Hostname(), Path: "/"},
	})

	body, status, err := c.Get(t.Context(), server.URL+"/a")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
	if receivedCookie != "user_session=secret" {
		t.Fatalf("Cookie = %q, want %q", receivedCookie, "user_session=secret")
	}
}
