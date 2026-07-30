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
	"github.com/sudosubin/gh-attach/internal/github/web"
	"golang.org/x/sync/errgroup"
)

type Request struct {
	FilePaths       []string
	Repo            string
	Browser         string
	Profile         string
	CookieStorePath string
	SessionToken    string
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

	var ghService *rest.Service
	getGHService := func() (*rest.Service, error) {
		if ghService != nil {
			return ghService, nil
		}
		service, err := rest.NewService(repoSpec.Host, nil)
		if err != nil {
			return nil, fmt.Errorf("init gh api service: %w", err)
		}
		ghService = service
		return ghService, nil
	}

	repo := rest.Repository{Host: repoSpec.Host, Owner: repoSpec.Owner, Name: repoSpec.Name}
	var session web.Session
	if req.SessionToken != "" {
		session, err = newTokenSession(repo.Host, req.SessionToken)
		if err != nil {
			return nil, err
		}
	} else {
		sources, resolveErr := cookies.ResolveSources(cookies.ResolveInput{
			Browser:         req.Browser,
			Profile:         req.Profile,
			CookieStorePath: req.CookieStorePath,
		})
		if resolveErr != nil {
			return nil, resolveErr
		}
		api, apiErr := getGHService()
		if apiErr != nil {
			return nil, apiErr
		}
		resolved, resolveErr := NewSessionResolver(
			NewConfigLoginResolver(api),
			NewCookieResolver(s.providers, req.Verbose, s.stderr),
		).Resolve(ctx, repo.Host, sources)
		if resolveErr != nil {
			return nil, resolveErr
		}
		session = web.Session{
			Cookies:   resolved.Session.Cookies,
			UserAgent: resolved.Session.UserAgent,
		}
	}

	uploader, err := attachments.NewUploader(repo.Host, session, nil)
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

	repositoryID := refererPage.Meta.RepositoryID
	if repositoryID == 0 {
		api, apiErr := getGHService()
		if apiErr != nil {
			return nil, apiErr
		}
		repositoryID, err = NewPageRepositoryIDResolver(api).RepositoryID(refererPage, repo.Owner, repo.Name)
		if err != nil {
			return nil, fmt.Errorf("resolve repository: %w", err)
		}
	}

	return uploadFiles(req.FilePaths, func(filePath string) (attachments.Asset, error) {
		return uploader.Upload(ctx, filePath, refererPage, repositoryID)
	})
}

const maxConcurrentUploads = 2

func uploadFiles(filePaths []string, upload func(string) (attachments.Asset, error)) ([]attachments.Asset, error) {
	results := make([]attachments.Asset, len(filePaths))
	errs := make([]error, len(filePaths))
	var uploads errgroup.Group
	uploads.SetLimit(maxConcurrentUploads)
	for i, filePath := range filePaths {
		uploads.Go(func() error {
			results[i], errs[i] = upload(filePath)
			if errs[i] != nil && len(filePaths) > 1 {
				errs[i] = fmt.Errorf("%s: %w", filePath, errs[i])
			}
			return errs[i]
		})
	}
	if err := uploads.Wait(); err == nil {
		return results, nil
	}

	assets := make([]attachments.Asset, 0, len(filePaths))
	var uploadErrs []error
	for i := range filePaths {
		if errs[i] != nil {
			uploadErrs = append(uploadErrs, errs[i])
			continue
		}
		assets = append(assets, results[i])
	}
	return assets, errors.Join(uploadErrs...)
}
