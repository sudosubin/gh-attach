package ghapi

import (
	"errors"
	"testing"
)

func TestResolveRepositorySpec_RepoNameOnlyUsesCurrentLogin(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	calledHost := ""
	parsed, err := resolveRepositorySpec("nix-skills", "", func(host string) (string, error) {
		calledHost = host
		return "sudosubin", nil
	})
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if calledHost != "github.com" {
		t.Fatalf("currentLogin host = %q, want %q", calledHost, "github.com")
	}
	if parsed.Host != "github.com" || parsed.Owner != "sudosubin" || parsed.Name != "nix-skills" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_RepoNameOnlyUsesHostnameArg(t *testing.T) {
	calledHost := ""
	parsed, err := resolveRepositorySpec("nix-skills", "github.example.com", func(host string) (string, error) {
		calledHost = host
		return "sudosubin", nil
	})
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if calledHost != "github.example.com" {
		t.Fatalf("currentLogin host = %q, want %q", calledHost, "github.example.com")
	}
	if parsed.Host != "github.example.com" || parsed.Owner != "sudosubin" || parsed.Name != "nix-skills" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_RepoNameOnlyPropagatesLoginError(t *testing.T) {
	wantErr := errors.New("boom")
	_, err := resolveRepositorySpec("nix-skills", "github.com", func(host string) (string, error) {
		return "", wantErr
	})
	if err == nil {
		t.Fatalf("resolveRepositorySpec() error = nil, want non-nil")
	}
}

func TestResolveRepositorySpec_OwnerRepoDoesNotCallCurrentLogin(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	called := false
	parsed, err := resolveRepositorySpec("octocat/hello", "", func(host string) (string, error) {
		called = true
		return "", nil
	})
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if called {
		t.Fatalf("currentLogin should not be called for owner/repo input")
	}
	if parsed.Owner != "octocat" || parsed.Name != "hello" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_HostnameOverridesRepoHost(t *testing.T) {
	called := false
	parsed, err := resolveRepositorySpec("github.com/octocat/hello", "ghe.example.com", func(host string) (string, error) {
		called = true
		return "", nil
	})
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if called {
		t.Fatalf("currentLogin should not be called for host/owner/repo input")
	}
	if parsed.Host != "ghe.example.com" {
		t.Fatalf("parsed.Host = %q, want %q", parsed.Host, "ghe.example.com")
	}
}

func TestResolveRepositorySpec_EmptyRepoUsesCurrentRepository(t *testing.T) {
	t.Setenv("GH_REPO", "octocat/hello")
	t.Setenv("GH_HOST", "github.com")

	parsed, err := resolveRepositorySpec("", "", func(host string) (string, error) {
		return "", nil
	})
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if parsed.Host != "github.com" || parsed.Owner != "octocat" || parsed.Name != "hello" {
		t.Fatalf("parsed = %#v", parsed)
	}
}
