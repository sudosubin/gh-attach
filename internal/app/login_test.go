package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/config"
)

func TestLoginForToken(t *testing.T) {
	const multiAccount = `
hosts:
  github.com:
    user: sudosubin
    users:
      sudosubin:
        oauth_token: tok-sudo
      other:
        oauth_token: tok-other
`
	const legacy = `
hosts:
  github.com:
    user: sudosubin
    oauth_token: tok-legacy
`
	const keyringOnly = `
hosts:
  github.com:
    user: sudosubin
    users:
      sudosubin:
`
	const enterprise = `
hosts:
  github.example.com:
    user: octocat
    users:
      octocat:
        oauth_token: tok-octocat
`

	tests := []struct {
		name      string
		yaml      string
		host      string
		token     string
		wantLogin string
		wantOK    bool
	}{
		{
			name:      "active user's stored token matches",
			yaml:      multiAccount,
			host:      "github.com",
			token:     "tok-sudo",
			wantLogin: "sudosubin",
			wantOK:    true,
		},
		{
			name:      "env override token belongs to a non-active account",
			yaml:      multiAccount,
			host:      "github.com",
			token:     "tok-other",
			wantLogin: "other",
			wantOK:    true,
		},
		{
			name:   "foreign token matches no account",
			yaml:   multiAccount,
			host:   "github.com",
			token:  "tok-foreign",
			wantOK: false,
		},
		{
			name:      "legacy top-level oauth_token",
			yaml:      legacy,
			host:      "github.com",
			token:     "tok-legacy",
			wantLogin: "sudosubin",
			wantOK:    true,
		},
		{
			name:   "keyring-only account (no stored token) never matches",
			yaml:   keyringOnly,
			host:   "github.com",
			token:  "tok-sudo",
			wantOK: false,
		},
		{
			name:   "empty token",
			yaml:   multiAccount,
			host:   "github.com",
			token:  "",
			wantOK: false,
		},
		{
			name:      "enterprise host",
			yaml:      enterprise,
			host:      "github.example.com",
			token:     "tok-octocat",
			wantLogin: "octocat",
			wantOK:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ReadFromString(tt.yaml)
			login, ok := loginForToken(cfg, tt.host, tt.token)
			if ok != tt.wantOK {
				t.Fatalf("loginForToken() ok = %v, want %v", ok, tt.wantOK)
			}
			if login != tt.wantLogin {
				t.Fatalf("loginForToken() login = %q, want %q", login, tt.wantLogin)
			}
		})
	}
}
