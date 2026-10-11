package app

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/sudosubin/gh-attach/internal/github/rest"
)

type apiRepositoryResolver interface {
	ResolveRepository(owner string, name string) (rest.Repository, error)
}

func resolveRepositorySpec(repoArg string) (repository.Repository, error) {
	repoArg = strings.TrimSpace(repoArg)
	if repoArg != "" {
		return repository.Parse(repoArg)
	}
	return repository.Current()
}

func resolveRepositoryID(pageID int64, repo rest.Repository, api apiRepositoryResolver) (int64, error) {
	if pageID > 0 {
		return pageID, nil
	}
	if api == nil {
		service, err := rest.NewService(repo.Host, nil)
		if err != nil {
			return 0, fmt.Errorf("init gh api service: %w", err)
		}
		api = service
	}
	resolved, err := api.ResolveRepository(repo.Owner, repo.Name)
	if err != nil {
		return 0, fmt.Errorf("resolve repository: %w", err)
	}
	return resolved.ID, nil
}
