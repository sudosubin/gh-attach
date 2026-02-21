package ghapi

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

type Repository struct {
	Host  string
	Owner string
	Name  string
	ID    int64
}

func (r Repository) FullName() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

func (s *Service) ResolveRepository(owner string, name string) (Repository, error) {
	var repoResp struct {
		ID int64 `json:"id"`
	}
	if err := s.client.Get(fmt.Sprintf("repos/%s/%s", owner, name), &repoResp); err != nil {
		return Repository{}, err
	}

	return Repository{
		Host:  s.host,
		Owner: owner,
		Name:  name,
		ID:    repoResp.ID,
	}, nil
}

func (s *Service) CurrentLogin() (string, error) {
	var userResp struct {
		Login string `json:"login"`
	}
	if err := s.client.Get("user", &userResp); err != nil {
		return "", err
	}
	if userResp.Login == "" {
		return "", fmt.Errorf("failed to resolve current GitHub login")
	}

	return userResp.Login, nil
}

func ResolveRepository(repoArg string) (Repository, error) {
	parsed, err := resolveRepositorySpec(repoArg)
	if err != nil {
		return Repository{}, err
	}

	svc, err := NewService(parsed.Host, nil)
	if err != nil {
		return Repository{}, err
	}

	return svc.ResolveRepository(parsed.Owner, parsed.Name)
}

func ResolveRepositorySpec(repoArg string) (repository.Repository, error) {
	return resolveRepositorySpec(repoArg)
}

func resolveRepositorySpec(repoArg string) (repository.Repository, error) {
	repoArg = strings.TrimSpace(repoArg)

	if repoArg != "" {
		return repository.Parse(repoArg)
	}

	return repository.Current()
}

func CurrentLogin(host string) (string, error) {
	svc, err := NewService(host, nil)
	if err != nil {
		return "", err
	}

	return svc.CurrentLogin()
}
