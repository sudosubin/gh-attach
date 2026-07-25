package app

import (
	"github.com/sudosubin/gh-attach/internal/github/attachments"
	"github.com/sudosubin/gh-attach/internal/github/rest"
)

// RepositoryIDResolver resolves the numeric repository_id the upload API needs.
type RepositoryIDResolver interface {
	RepositoryID(refererPage *attachments.RefererPage, owner string, name string) (int64, error)
}

// apiRepositoryResolver is the REST fallback, satisfied by rest.Service.ResolveRepository.
type apiRepositoryResolver interface {
	ResolveRepository(owner string, name string) (rest.Repository, error)
}

type pageRepositoryIDResolver struct {
	api apiRepositoryResolver
}

func NewPageRepositoryIDResolver(api apiRepositoryResolver) RepositoryIDResolver {
	return pageRepositoryIDResolver{api: api}
}

func (r pageRepositoryIDResolver) RepositoryID(refererPage *attachments.RefererPage, owner string, name string) (int64, error) {
	if id := refererPage.Meta.RepositoryID; id > 0 {
		return id, nil
	}
	repo, err := r.api.ResolveRepository(owner, name)
	if err != nil {
		return 0, err
	}
	return repo.ID, nil
}
