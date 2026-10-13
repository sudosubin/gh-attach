package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/github/rest"
)

type fakeAPIRepository struct {
	repo              rest.Repository
	err               error
	calls             int
	gotOwner, gotName string
}

func (f *fakeAPIRepository) ResolveRepository(owner string, name string) (rest.Repository, error) {
	f.calls++
	f.gotOwner, f.gotName = owner, name
	return f.repo, f.err
}

func TestResolveRepositorySpec_RejectsRepoOnly(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	_, err := resolveRepositorySpec("repo")
	if err == nil {
		t.Fatalf("resolveRepositorySpec() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "[HOST/]OWNER/REPO") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "[HOST/]OWNER/REPO")
	}
}

func TestResolveRepositorySpec_OwnerRepo(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")

	parsed, err := resolveRepositorySpec("owner/repo")
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if parsed.Owner != "owner" || parsed.Name != "repo" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_HostOwnerRepo(t *testing.T) {
	parsed, err := resolveRepositorySpec("github.example.com/owner/repo")
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if parsed.Host != "github.example.com" || parsed.Owner != "owner" || parsed.Name != "repo" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositorySpec_EmptyRepoUsesCurrentRepository(t *testing.T) {
	t.Setenv("GH_REPO", "owner/repo")
	t.Setenv("GH_HOST", "github.com")

	parsed, err := resolveRepositorySpec("")
	if err != nil {
		t.Fatalf("resolveRepositorySpec() error = %v", err)
	}
	if parsed.Host != "github.com" || parsed.Owner != "owner" || parsed.Name != "repo" {
		t.Fatalf("parsed = %#v", parsed)
	}
}

func TestResolveRepositoryID(t *testing.T) {
	tests := []struct {
		name         string
		pageID       int64
		api          *fakeAPIRepository
		wantID       int64
		wantErr      bool
		wantAPICalls int
	}{
		{
			name:   "page-parsed id is preferred over the API",
			pageID: 261246700,
			api:    &fakeAPIRepository{repo: rest.Repository{ID: 999}},
			wantID: 261246700,
		},
		{
			name:         "falls back to the API when the page has no id",
			pageID:       0,
			api:          &fakeAPIRepository{repo: rest.Repository{ID: 42}},
			wantID:       42,
			wantAPICalls: 1,
		},
		{
			name:         "propagates the API error",
			pageID:       0,
			api:          &fakeAPIRepository{err: fmt.Errorf("boom")},
			wantErr:      true,
			wantAPICalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := resolveRepositoryID(tt.pageID, rest.Repository{
				Host: "github.com", Owner: "owner", Name: "repo",
			}, tt.api)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveRepositoryID() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRepositoryID() error = %v", err)
			}
			if id != tt.wantID {
				t.Fatalf("resolveRepositoryID() = %d, want %d", id, tt.wantID)
			}
			if tt.api.calls != tt.wantAPICalls {
				t.Fatalf("api.calls = %d, want %d", tt.api.calls, tt.wantAPICalls)
			}
			if tt.wantAPICalls > 0 && (tt.api.gotOwner != "owner" || tt.api.gotName != "repo") {
				t.Fatalf("api called with (%q, %q), want (%q, %q)", tt.api.gotOwner, tt.api.gotName, "owner", "repo")
			}
		})
	}
}
