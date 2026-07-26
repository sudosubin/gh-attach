package app

import (
	"fmt"
	"testing"

	"github.com/sudosubin/gh-attach/internal/github/attachments"
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

func TestPageRepositoryIDResolver(t *testing.T) {
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
			refererPage := &attachments.RefererPage{}
			refererPage.Meta.RepositoryID = tt.pageID

			id, err := NewPageRepositoryIDResolver(tt.api).RepositoryID(refererPage, "owner", "repo")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("RepositoryID() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("RepositoryID() error = %v", err)
			}
			if id != tt.wantID {
				t.Fatalf("RepositoryID() = %d, want %d", id, tt.wantID)
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
