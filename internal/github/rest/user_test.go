package rest

import (
	"fmt"
	"testing"
)

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
