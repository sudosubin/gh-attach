package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/config"
	"github.com/sudosubin/gh-attach/internal/github/rest"
)

// lazyAPILoginResolver defers building a REST client until Login() actually
// needs the API fallback, so hosts.yml/env-token users — who resolve via
// loginFromLocalToken and never reach CurrentLogin — don't pay for
// constructing a client they won't use.
type lazyAPILoginResolver struct {
	host string
}

func (r lazyAPILoginResolver) CurrentLogin() (string, error) {
	api, err := rest.NewService(r.host, nil)
	if err != nil {
		return "", fmt.Errorf("init gh api service: %w", err)
	}
	return api.CurrentLogin()
}

// loginFromLocalToken resolves the login from local gh config; an empty token means keyring/absent -> API fallback.
func loginFromLocalToken(host string) (string, bool) {
	token, _ := auth.TokenFromEnvOrConfig(host)
	if token == "" {
		return "", false
	}

	cfg, err := config.Read(nil)
	if err != nil {
		return "", false
	}

	return loginForToken(cfg, auth.NormalizeHostname(host), token)
}

// loginForToken finds the account owning token, matched by value so a stale hosts.yml can't mislead.
func loginForToken(cfg *config.Config, host, token string) (string, bool) {
	if token == "" {
		return "", false
	}

	// Trust the active user only when its stored token matches.
	if active, _ := cfg.Get([]string{"hosts", host, "user"}); active != "" {
		if s, _ := cfg.Get([]string{"hosts", host, "users", active, "oauth_token"}); s != "" && s == token {
			return active, true
		}
		// Legacy top-level slot (pre multi-account hosts.yml).
		if s, _ := cfg.Get([]string{"hosts", host, "oauth_token"}); s != "" && s == token {
			return active, true
		}
	}

	// Env-overridden token: find its owner among the stored accounts.
	logins, _ := cfg.Keys([]string{"hosts", host, "users"})
	for _, login := range logins {
		if s, _ := cfg.Get([]string{"hosts", host, "users", login, "oauth_token"}); s != "" && s == token {
			return login, true
		}
	}

	return "", false
}
