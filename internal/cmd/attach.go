package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cmdutil"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/formatting"
	"github.com/sudosubin/gh-attach/internal/ghapi"
	"github.com/sudosubin/gh-attach/internal/upload"
)

type AttachOptions struct {
	FilePath        string
	Repo            string
	Browser         string
	Profile         string
	CookieStorePath string
	JSON            cmdutil.JSONFlags
	Verbose         bool
}

func NewCmdAttach(runF func(*AttachOptions) error) *cobra.Command {
	opts := &AttachOptions{}

	cmd := &cobra.Command{
		Use:   "<file>",
		Short: "Upload a file to GitHub user-attachments",
		Long:  "Upload a file to GitHub user-attachments.\n\nFor more information about output formatting flags, see `gh help formatting`.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.FilePath = args[0]
			if runF != nil {
				return runF(opts)
			}
			return attachRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "[HOST/]OWNER/REPO")
	cmd.Flags().StringVar(&opts.Browser, "browser", "", "Browser to use (auto|chrome|chromium|edge|firefox|safari|brave|vivaldi|opera)")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "Browser profile name or path")
	cmd.Flags().StringVar(&opts.CookieStorePath, "cookie-store-path", "", "Cookie store file path")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output")

	cmdutil.AddJSONFlags(cmd, &opts.JSON, formatting.AvailableJSONFields())

	return cmd
}

func attachRun(opts *AttachOptions) error {
	if _, err := os.Stat(opts.FilePath); err != nil {
		return fmt.Errorf("file: %w", err)
	}

	repoSpec, err := ghapi.ResolveRepositorySpec(strings.TrimSpace(opts.Repo))
	if err != nil {
		return fmt.Errorf("resolve repository spec: %w", err)
	}

	ghService, err := ghapi.NewService(repoSpec.Host, nil)
	if err != nil {
		return fmt.Errorf("init gh api service: %w", err)
	}

	repo, err := ghService.ResolveRepository(repoSpec.Owner, repoSpec.Name)
	if err != nil {
		return fmt.Errorf("resolve repository: %w", err)
	}

	ghLogin, err := ghService.CurrentLogin()
	if err != nil {
		return fmt.Errorf("resolve current login: %w", err)
	}

	sources, err := cookies.ResolveSources(cookies.ResolveInput{
		Browser:         opts.Browser,
		Profile:         opts.Profile,
		CookieStorePath: opts.CookieStorePath,
	})
	if err != nil {
		return err
	}

	providers := browserprovider.NewDefaultRegistry()

	session, selectedSource, selectedProvider, err := resolveCookies(context.Background(), repo.Host, ghLogin, sources, providers, opts.Verbose)
	if err != nil {
		return err
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "selected source: browser=%s profile=%q cookie_store_path=%q provider=%s\n", selectedSource.Browser, selectedSource.Profile, selectedSource.CookieStorePath, selectedProvider)
	}

	uploader, err := upload.NewUploader(repo.Host, repo.ID, session, nil)
	if err != nil {
		return fmt.Errorf("init uploader: %w", err)
	}

	refererPage, err := uploader.ResolveRefererPage(
		context.Background(),
		[]upload.RefererPageFetcher{
			upload.NewIssueNewPageFetcher(repo.Host, repo.FullName()),
			upload.NewLatestCommitPageFetcher(repo.Host, repo.Owner, repo.Name, ghService),
		},
	)
	if err != nil {
		return err
	}

	asset, err := uploader.Upload(context.Background(), opts.FilePath, refererPage)
	if err != nil {
		return err
	}

	return formatting.WriteAsset(os.Stdout, asset, formatting.Options{
		JSONFlagSet: opts.JSON.Enabled,
		JSONFields:  opts.JSON.Fields,
		JQ:          opts.JSON.Filter,
		Template:    opts.JSON.Template,
	})
}

func resolveCookies(
	ctx context.Context,
	host string,
	ghLogin string,
	sources []cookies.Source,
	providers map[cookies.Browser]browserprovider.BrowserProvider,
	verbose bool,
) (browserprovider.BrowserSession, cookies.Source, string, error) {
	attempts := 0
	for idx, source := range sources {
		expanded := cookies.ExpandSource(source)
		for _, candidate := range expanded {
			candidate = applyDefaultProfile(candidate)
			attempts++

			provider, ok := providers[candidate.Browser]
			if !ok {
				if verbose {
					fmt.Fprintf(os.Stderr, "source[%d]: browser=%s provider=none missing\n", idx, candidate.Browser)
				}
				continue
			}

			backendName := provider.BackendName()
			session, err := provider.Load(ctx, host, candidate)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "source[%d]: browser=%s profile=%q cookie_store_path=%q provider=%s error=%v\n", idx, candidate.Browser, candidate.Profile, candidate.CookieStorePath, backendName, err)
				}
				continue
			}

			dotcomUsers := cookieValuesByNameForHost(session.Cookies, "dotcom_user", host)
			if len(dotcomUsers) == 0 {
				if verbose {
					fmt.Fprintf(os.Stderr, "source[%d]: browser=%s provider=%s skipped (dotcom_user missing)\n", idx, candidate.Browser, backendName)
				}
				continue
			}
			if !containsFold(dotcomUsers, ghLogin) {
				if verbose {
					fmt.Fprintf(os.Stderr, "source[%d]: browser=%s provider=%s skipped (dotcom_user=%q != gh_login=%q)\n", idx, candidate.Browser, backendName, strings.Join(dotcomUsers, ","), ghLogin)
				}
				continue
			}

			if verbose {
				fmt.Fprintf(os.Stderr, "source[%d]: browser=%s provider=%s matched dotcom_user=%q\n", idx, candidate.Browser, backendName, ghLogin)
			}
			return session, candidate, backendName, nil
		}
	}

	return browserprovider.BrowserSession{}, cookies.Source{}, "", fmt.Errorf("failed to resolve usable cookie source from %d attempt(s)", attempts)
}

func cookieValuesByNameForHost(in []*http.Cookie, name string, host string) []string {
	host = strings.TrimSpace(host)
	if host == "" {
		return cookieValuesByName(in, name)
	}

	u := &url.URL{Scheme: "https", Host: host, Path: "/"}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return cookieValuesByName(in, name)
	}
	jar.SetCookies(u, in)

	return cookieValuesByName(jar.Cookies(u), name)
}

func cookieValuesByName(in []*http.Cookie, name string) []string {
	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, c := range in {
		if c == nil {
			continue
		}
		if c.Name != name || c.Value == "" {
			continue
		}
		if _, ok := seen[c.Value]; ok {
			continue
		}
		seen[c.Value] = struct{}{}
		out = append(out, c.Value)
	}
	return out
}

func containsFold(values []string, target string) bool {
	for _, v := range values {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}

func applyDefaultProfile(source cookies.Source) cookies.Source {
	if strings.TrimSpace(source.Profile) != "" || strings.TrimSpace(source.CookieStorePath) != "" {
		return source
	}

	switch source.Browser {
	case cookies.BrowserChrome,
		cookies.BrowserChromium,
		cookies.BrowserEdge,
		cookies.BrowserBrave,
		cookies.BrowserVivaldi,
		cookies.BrowserOpera:
		source.Profile = "Default"
	}

	return source
}
