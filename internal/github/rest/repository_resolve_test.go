package rest

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// stubRESTGetter implements RESTGetter, unmarshalling a canned body or returning a canned error.
type stubRESTGetter struct {
	body string
	err  error
}

func (s stubRESTGetter) Get(_ string, data any) error {
	if s.err != nil {
		return s.err
	}
	return json.Unmarshal([]byte(s.body), data)
}

func TestResolveRepository(t *testing.T) {
	tests := []struct {
		name    string
		stub    stubRESTGetter
		wantID  int64
		wantErr bool
	}{
		{
			name:   "valid id",
			stub:   stubRESTGetter{body: `{"id":261246700}`},
			wantID: 261246700,
		},
		{
			name:    "zero id",
			stub:    stubRESTGetter{body: `{"id":0}`},
			wantErr: true,
		},
		{
			name:    "missing id",
			stub:    stubRESTGetter{body: `{}`},
			wantErr: true,
		},
		{
			name:    "negative id",
			stub:    stubRESTGetter{body: `{"id":-5}`},
			wantErr: true,
		},
		{
			name:    "client error propagated",
			stub:    stubRESTGetter{err: fmt.Errorf("boom")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService("github.com", tt.stub)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}

			repo, err := svc.ResolveRepository("octocat", "hello")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolveRepository() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveRepository() error = %v", err)
			}
			if repo.ID != tt.wantID {
				t.Fatalf("ResolveRepository() ID = %d, want %d", repo.ID, tt.wantID)
			}
			if repo.Owner != "octocat" || repo.Name != "hello" {
				t.Fatalf("ResolveRepository() repo = %#v", repo)
			}
		})
	}
}

func TestCurrentLogin(t *testing.T) {
	tests := []struct {
		name    string
		stub    stubRESTGetter
		want    string
		wantErr bool
	}{
		{name: "valid login", stub: stubRESTGetter{body: `{"login":"octocat"}`}, want: "octocat"},
		{name: "empty login", stub: stubRESTGetter{body: `{}`}, wantErr: true},
		{name: "client error propagated", stub: stubRESTGetter{err: fmt.Errorf("boom")}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService("github.com", tt.stub)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}

			login, err := svc.CurrentLogin()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("CurrentLogin() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("CurrentLogin() error = %v", err)
			}
			if login != tt.want {
				t.Fatalf("CurrentLogin() = %q, want %q", login, tt.want)
			}
		})
	}
}

func TestResolveRepository_ErrorMentionsRepo(t *testing.T) {
	svc, err := NewService("github.com", stubRESTGetter{body: `{"id":0}`})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = svc.ResolveRepository("octocat", "hello")
	if err == nil {
		t.Fatalf("ResolveRepository() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "octocat/hello") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "octocat/hello")
	}
}
