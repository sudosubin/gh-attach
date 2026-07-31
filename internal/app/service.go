package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

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

// Service orchestrates GitHub user-attachment flows.
type Service struct {
	providers     map[cookies.Browser]browserprovider.BrowserProvider
	stderr        io.Writer
	httpClient    *http.Client
	loginResolver LoginResolver
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

	repo := rest.Repository{Host: repoSpec.Host, Owner: repoSpec.Owner, Name: repoSpec.Name}
	var session web.Session
	if req.SessionToken != "" {
		session, err = newTokenSession(repo.Host, req.SessionToken)
	} else {
		session, err = s.resolveBrowserSession(ctx, repo.Host, cookies.ResolveInput{
			Browser:         req.Browser,
			Profile:         req.Profile,
			CookieStorePath: req.CookieStorePath,
		}, req.Verbose)
	}
	if err != nil {
		return nil, err
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
		api, apiErr := rest.NewService(repo.Host, nil)
		if apiErr != nil {
			return nil, fmt.Errorf("init gh api service: %w", apiErr)
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
