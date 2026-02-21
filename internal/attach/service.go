package attach

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/ghapi"
	"github.com/sudosubin/gh-attach/internal/upload"
)

type Request struct {
	FilePath        string
	Repo            string
	Browser         string
	Profile         string
	CookieStorePath string
	Verbose         bool
}

// Service orchestrates the GitHub user-attachment upload flow.
type Service struct {
	providers map[cookies.Browser]browserprovider.BrowserProvider
	stderr    io.Writer
}

func NewService(stderr io.Writer) *Service {
	return &Service{
		providers: browserprovider.NewDefaultRegistry(),
		stderr:    stderr,
	}
}

func (s *Service) Run(ctx context.Context, req Request) (upload.Asset, error) {
	if _, err := os.Stat(req.FilePath); err != nil {
		return upload.Asset{}, fmt.Errorf("file: %w", err)
	}

	repoSpec, err := ghapi.ResolveRepositorySpec(req.Repo)
	if err != nil {
		return upload.Asset{}, fmt.Errorf("resolve repository spec: %w", err)
	}

	ghService, err := ghapi.NewService(repoSpec.Host, nil)
	if err != nil {
		return upload.Asset{}, fmt.Errorf("init gh api service: %w", err)
	}

	repo, err := ghService.ResolveRepository(repoSpec.Owner, repoSpec.Name)
	if err != nil {
		return upload.Asset{}, fmt.Errorf("resolve repository: %w", err)
	}

	ghLogin, err := ghService.CurrentLogin()
	if err != nil {
		return upload.Asset{}, fmt.Errorf("resolve current login: %w", err)
	}

	sources, err := cookies.ResolveSources(cookies.ResolveInput{
		Browser:         req.Browser,
		Profile:         req.Profile,
		CookieStorePath: req.CookieStorePath,
	})
	if err != nil {
		return upload.Asset{}, err
	}

	resolver := NewCookieResolver(s.providers, req.Verbose, s.stderr)

	resolved, err := resolver.Resolve(ctx, repo.Host, ghLogin, sources)
	if err != nil {
		return upload.Asset{}, err
	}

	uploader, err := upload.NewUploader(repo.Host, repo.ID, resolved.Session, nil)
	if err != nil {
		return upload.Asset{}, fmt.Errorf("init uploader: %w", err)
	}

	refererPage, err := uploader.ResolveRefererPage(
		ctx,
		[]upload.RefererPageFetcher{
			upload.NewIssueNewPageFetcher(repo.Host, repo.FullName()),
			upload.NewLatestCommitPageFetcher(repo.Host, repo.Owner, repo.Name, ghService),
		},
	)
	if err != nil {
		return upload.Asset{}, err
	}

	return uploader.Upload(ctx, req.FilePath, refererPage)
}
