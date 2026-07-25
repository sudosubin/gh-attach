package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/attachments"
	"github.com/sudosubin/gh-attach/internal/github/rest"
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

func (s *Service) Run(ctx context.Context, req Request) (attachments.Asset, error) {
	if _, err := os.Stat(req.FilePath); err != nil {
		return attachments.Asset{}, fmt.Errorf("file: %w", err)
	}

	repoSpec, err := rest.ResolveRepositorySpec(req.Repo)
	if err != nil {
		return attachments.Asset{}, fmt.Errorf("resolve repository spec: %w", err)
	}

	ghService, err := rest.NewService(repoSpec.Host, nil)
	if err != nil {
		return attachments.Asset{}, fmt.Errorf("init gh api service: %w", err)
	}

	sources, err := cookies.ResolveSources(cookies.ResolveInput{
		Browser:         req.Browser,
		Profile:         req.Profile,
		CookieStorePath: req.CookieStorePath,
	})
	if err != nil {
		return attachments.Asset{}, err
	}

	repo := rest.Repository{Host: repoSpec.Host, Owner: repoSpec.Owner, Name: repoSpec.Name}

	sessions := NewSessionResolver(
		NewConfigLoginResolver(ghService),
		NewCookieResolver(s.providers, req.Verbose, s.stderr),
	)
	resolved, err := sessions.Resolve(ctx, repo.Host, sources)
	if err != nil {
		return attachments.Asset{}, err
	}

	uploader, err := attachments.NewUploader(repo.Host, resolved.Session, nil)
	if err != nil {
		return attachments.Asset{}, fmt.Errorf("init uploader: %w", err)
	}

	refererPage, err := uploader.ResolveRefererPage(
		ctx,
		[]attachments.RefererPageFetcher{
			attachments.NewIssueNewPageFetcher(repo.Host, repo.FullName()),
			attachments.NewLatestCommitPageFetcher(repo.Host, repo.Owner, repo.Name, ghService),
		},
	)
	if err != nil {
		return attachments.Asset{}, err
	}

	repositoryID, err := NewPageRepositoryIDResolver(ghService).RepositoryID(refererPage, repo.Owner, repo.Name)
	if err != nil {
		return attachments.Asset{}, fmt.Errorf("resolve repository: %w", err)
	}

	return uploader.Upload(ctx, req.FilePath, refererPage, repositoryID)
}
