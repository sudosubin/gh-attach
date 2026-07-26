package rest

import (
	"strings"
	"testing"
)

func TestResolveRepositorySpec_RejectsRepoOnly(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	_, err := ResolveRepositorySpec("repo")
	if err == nil {
		t.Fatalf("ResolveRepositorySpec() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "[HOST/]OWNER/REPO") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "[HOST/]OWNER/REPO")
	}
}

func TestResolveRepositorySpec_OwnerRepo(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	parsed, err := ResolveRepositorySpec("owner/repo")
	if err != nil {
		t.Fatalf("ResolveRepositorySpec() error = %v", err)
	}
	if parsed.Owner != "owner" || parsed.Name != "repo" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_HostOwnerRepo(t *testing.T) {
	parsed, err := ResolveRepositorySpec("github.example.com/owner/repo")
	if err != nil {
		t.Fatalf("ResolveRepositorySpec() error = %v", err)
	}
	if parsed.Host != "github.example.com" || parsed.Owner != "owner" || parsed.Name != "repo" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_EmptyRepoUsesCurrentRepository(t *testing.T) {
	t.Setenv("GH_REPO", "owner/repo")
	t.Setenv("GH_HOST", "github.com")

	parsed, err := ResolveRepositorySpec("")
	if err != nil {
		t.Fatalf("ResolveRepositorySpec() error = %v", err)
	}
	if parsed.Host != "github.com" || parsed.Owner != "owner" || parsed.Name != "repo" {
		t.Fatalf("parsed = %#v", parsed)
	}
}
