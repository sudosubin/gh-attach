package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/attachments"
	"github.com/sudosubin/gh-attach/internal/github/rest"
)

type Request struct {
	FilePaths       []string
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

func (s *Service) Run(ctx context.Context, req Request) ([]attachments.Asset, error) {
	if len(req.FilePaths) == 0 {
		return nil, errors.New("no files to upload")
	}

	repoSpec, err := rest.ResolveRepositorySpec(req.Repo)
	if err != nil {
		return nil, fmt.Errorf("resolve repository spec: %w", err)
	}

	ghService, err := rest.NewService(repoSpec.Host, nil)
	if err != nil {
		return nil, fmt.Errorf("init gh api service: %w", err)
	}

	sources, err := cookies.ResolveSources(cookies.ResolveInput{
		Browser:         req.Browser,
		Profile:         req.Profile,
		CookieStorePath: req.CookieStorePath,
	})
	if err != nil {
		return nil, err
	}

	repo := rest.Repository{Host: repoSpec.Host, Owner: repoSpec.Owner, Name: repoSpec.Name}

	sessions := NewSessionResolver(
		NewConfigLoginResolver(ghService),
		NewCookieResolver(s.providers, req.Verbose, s.stderr),
	)
	resolved, err := sessions.Resolve(ctx, repo.Host, sources)
	if err != nil {
		return nil, err
	}

	uploader, err := attachments.NewUploader(repo.Host, resolved.Session, nil)
	if err != nil {
		return nil, fmt.Errorf("init uploader: %w", err)
	}

	refererPage, err := uploader.ResolveRefererPage(
		ctx,
		[]attachments.RefererPageFetcher{
			attachments.NewIssueNewPageFetcher(repo.Host, repo.FullName()),
			attachments.NewCommitHeadPageFetcher(repo.Host, repo.FullName()),
		},
	)
	if err != nil {
		return nil, err
	}

	repositoryID, err := NewPageRepositoryIDResolver(ghService).RepositoryID(refererPage, repo.Owner, repo.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w", err)
	}

	return uploadFiles(req.FilePaths, func(filePath string) (attachments.Asset, error) {
		return uploader.Upload(ctx, filePath, refererPage, repositoryID)
	})
}

func uploadFiles(filePaths []string, upload func(string) (attachments.Asset, error)) ([]attachments.Asset, error) {
	assets := make([]attachments.Asset, 0, len(filePaths))
	var errs []error
	for _, filePath := range filePaths {
		asset, err := upload(filePath)
		if err != nil {
			if len(filePaths) > 1 {
				err = fmt.Errorf("%s: %w", filePath, err)
			}
			errs = append(errs, err)
			continue
		}
		assets = append(assets, asset)
	}
	return assets, errors.Join(errs...)
}
