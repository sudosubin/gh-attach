package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/sudosubin/gh-attach/internal/browserprovider"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/github/attachments"
	"github.com/sudosubin/gh-attach/internal/github/rest"
	"github.com/sudosubin/gh-attach/internal/github/web"
	"golang.org/x/sync/errgroup"
)

type Request struct {
	FilePaths            []string
	Repo                 string
	Browser              string
	Profile              string
	CookieStorePath      string
	SessionToken         string
	UseBrowser           bool
	RequireAssetMetadata bool
	Verbose              bool
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

	repoSpec, err := resolveRepositorySpec(req.Repo)
	if err != nil {
		return nil, fmt.Errorf("resolve repository spec: %w", err)
	}

	repo := rest.Repository{Host: repoSpec.Host, Owner: repoSpec.Owner, Name: repoSpec.Name}

	var logMu sync.Mutex
	logf := func(format string, args ...any) {
		if req.Verbose && s.stderr != nil {
			logMu.Lock()
			defer logMu.Unlock()
			fmt.Fprintf(s.stderr, format, args...)
		}
	}
	cookieUpload := sync.OnceValues(func() (func(string) (attachments.Asset, error), error) {
		// Cookie resolution also logs; serialize it with per-file fallback messages.
		if req.Verbose {
			logMu.Lock()
			defer logMu.Unlock()
		}
		return s.prepareCookieUpload(ctx, repo, req)
	})

	var token string
	if !auth.IsEnterprise(repo.Host) && req.SessionToken == "" && !req.UseBrowser &&
		req.Browser == "" && req.Profile == "" && req.CookieStorePath == "" && !req.RequireAssetMetadata &&
		slices.ContainsFunc(req.FilePaths, attachments.SupportsTokenUpload) {
		token, _ = auth.TokenForHost(repo.Host)
		token = strings.TrimSpace(token)
	}
	tokenUpload := sync.OnceValues(func() (*attachments.TokenUploader, error) {
		logf("preparing token upload for %s\n", repo.FullName())
		return s.prepareTokenUpload(repo, token)
	})

	return uploadFiles(req.FilePaths, func(filePath string) (attachments.Asset, error) {
		if err := ctx.Err(); err != nil {
			return attachments.Asset{}, err
		}
		if _, err := attachments.StatUploadFile(filePath); err != nil {
			return attachments.Asset{}, err
		}
		var firstErr error
		if canAttemptTokenUpload(token) && attachments.SupportsTokenUpload(filePath) {
			uploader, err := tokenUpload()
			var asset attachments.Asset
			if err == nil {
				asset, err = uploader.Upload(ctx, filePath)
			}
			if err == nil {
				return asset, nil
			}
			if ctx.Err() != nil || !isAuthenticationError(err) {
				return attachments.Asset{}, err
			}
			firstErr = fmt.Errorf("token upload: %w", err)
			logf("%s: token authentication rejected; falling back to browser cookies\n", filePath)
		}
		upload, err := cookieUpload()
		if err == nil {
			var asset attachments.Asset
			asset, err = upload(filePath)
			if err == nil {
				return asset, nil
			}
		}
		if firstErr != nil {
			err = errors.Join(firstErr, fmt.Errorf("browser cookie fallback: %w", err))
		}
		return attachments.Asset{}, err
	})
}

func (s *Service) prepareTokenUpload(repo rest.Repository, token string) (*attachments.TokenUploader, error) {
	var transport http.RoundTripper
	if s.httpClient != nil {
		transport = s.httpClient.Transport
	}
	client, err := rest.NewClient(repo.Host, token, transport)
	if err != nil {
		return nil, fmt.Errorf("init gh api client: %w", err)
	}
	service, err := rest.NewService(repo.Host, client)
	if err != nil {
		return nil, err
	}
	resolved, err := service.ResolveRepository(repo.Owner, repo.Name)
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w", err)
	}
	return attachments.NewTokenUploader(repo.Host, token, resolved.ID, s.httpClient)
}

func (s *Service) prepareCookieUpload(ctx context.Context, repo rest.Repository, req Request) (func(string) (attachments.Asset, error), error) {
	attachments.PrewarmUploadHostDNS(ctx, repo.Host)

	var session web.Session
	var err error
	if req.SessionToken != "" {
		session, err = newTokenSession(repo.Host, req.SessionToken)
	} else {
		session, err = s.resolveBrowserSession(ctx, repo.Host, cookieSourceInput{
			Browser:         req.Browser,
			Profile:         req.Profile,
			CookieStorePath: req.CookieStorePath,
		}, req.Verbose)
	}
	if err != nil {
		return nil, err
	}

	uploader, err := attachments.NewUploader(repo.Host, session, s.httpClient)
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

	repositoryID, err := resolveRepositoryID(refererPage.Meta.RepositoryID, repo, nil)
	if err != nil {
		return nil, err
	}

	return func(filePath string) (attachments.Asset, error) {
		return uploader.Upload(ctx, filePath, refererPage, repositoryID)
	}, nil
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
