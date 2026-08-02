package app

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type ResolvedCookies struct {
	Session      browserprovider.BrowserSession
	Source       cookies.Source
	ProviderName string
}

// CookieResolver finds the first browser session whose dotcom_user matches a GitHub login.
type CookieResolver struct {
	providers map[cookies.Browser]browserprovider.BrowserProvider
	verbose   bool
	stderr    io.Writer
}

func NewCookieResolver(
	providers map[cookies.Browser]browserprovider.BrowserProvider,
	verbose bool,
	stderr io.Writer,
) *CookieResolver {
	return &CookieResolver{
		providers: providers,
		verbose:   verbose,
		stderr:    stderr,
	}
}

// Resolve returns the first session matching ghLogin via dotcom_user. ghLogin
// is a thunk called only when a candidate needs comparing, so a concurrent
// login lookup (see SessionResolver.Resolve) overlaps and isn't awaited early.
func (r *CookieResolver) Resolve(ctx context.Context, host string, ghLogin func() (string, error), sources []cookies.Source) (ResolvedCookies, error) {
	attempts := 0

	for idx, source := range sources {
		for _, candidate := range cookies.ExpandSource(source) {
			candidate = cookies.ApplyDefaultProfile(candidate)
			attempts++

			provider, ok := r.providers[candidate.Browser]
			if !ok {
				r.logf("source[%d]: browser=%s provider=none missing\n", idx, candidate.Browser)
				continue
			}

			backendName := provider.BackendName()
			sessions, err := provider.Load(ctx, host, candidate)
			if err != nil {
				r.logf("source[%d]: browser=%s profile=%q cookie_store_path=%q provider=%s error=%v\n",
					idx, candidate.Browser, candidate.Profile, candidate.CookieStorePath, backendName, err)
				continue
			}

			for _, session := range sessions {
				dotcomUsers := cookies.ValuesForHost(session.Cookies, "dotcom_user", host)
				if len(dotcomUsers) == 0 {
					r.logf("source[%d]: browser=%s provider=%s profile=%q skipped (dotcom_user missing)\n",
						idx, candidate.Browser, backendName, session.Profile)
					continue
				}

				login, err := ghLogin()
				if err != nil {
					return ResolvedCookies{}, err
				}

				if !containsFold(dotcomUsers, login) {
					r.logf("source[%d]: browser=%s provider=%s profile=%q skipped (dotcom_user=%q != gh_login=%q)\n",
						idx, candidate.Browser, backendName, session.Profile, strings.Join(dotcomUsers, ","), login)
					continue
				}

				r.logf("source[%d]: browser=%s provider=%s profile=%q matched dotcom_user=%q\n",
					idx, candidate.Browser, backendName, session.Profile, login)
				r.logf("selected source: browser=%s profile=%q cookie_store_path=%q provider=%s\n",
					candidate.Browser, session.Profile, candidate.CookieStorePath, backendName)

				return ResolvedCookies{
					Session:      session,
					Source:       candidate,
					ProviderName: backendName,
				}, nil
			}
		}
	}

	return ResolvedCookies{}, fmt.Errorf("failed to resolve usable cookie source from %d attempt(s)", attempts)
}

func (r *CookieResolver) logf(format string, args ...any) {
	if r.verbose && r.stderr != nil {
		fmt.Fprintf(r.stderr, format, args...)
	}
}

func containsFold(values []string, target string) bool {
	return slices.ContainsFunc(values, func(v string) bool {
		return strings.EqualFold(v, target)
	})
}
