package ghapi

import (
	"strings"
	"testing"
)

func TestResolveRepositorySpec_RejectsRepoOnly(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	_, err := resolveRepositorySpec("nix-skills")
	if err == nil {
		t.Fatalf("resolveRepositorySpec() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "[HOST/]OWNER/REPO") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "[HOST/]OWNER/REPO")
	}
}

func TestResolveRepositorySpec_OwnerRepo(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	parsed, err := resolveRepositorySpec("octocat/hello")
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if parsed.Owner != "octocat" || parsed.Name != "hello" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_HostOwnerRepo(t *testing.T) {
	parsed, err := resolveRepositorySpec("github.example.com/octocat/hello")
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if parsed.Host != "github.example.com" || parsed.Owner != "octocat" || parsed.Name != "hello" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_EmptyRepoUsesCurrentRepository(t *testing.T) {
	t.Setenv("GH_REPO", "octocat/hello")
	t.Setenv("GH_HOST", "github.com")

	parsed, err := resolveRepositorySpec("")
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if parsed.Host != "github.com" || parsed.Owner != "octocat" || parsed.Name != "hello" {
		t.Fatalf("parsed = %#v", parsed)
	}
}
