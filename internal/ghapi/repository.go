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

func ResolveRepository(repoArg string) (Repository, error) {
	parsed, err := resolveRepositorySpec(repoArg)
	if err != nil {
		return Repository{}, err
	}

	client, err := newRESTClient(parsed.Host)
	if err != nil {
		return Repository{}, err
	}

	var repoResp struct {
		ID int64 `json:"id"`
	}
	if err := client.Get(fmt.Sprintf("repos/%s/%s", parsed.Owner, parsed.Name), &repoResp); err != nil {
		return Repository{}, err
	}

	return Repository{
		Host:  parsed.Host,
		Owner: parsed.Owner,
		Name:  parsed.Name,
		ID:    repoResp.ID,
	}, nil
}

func resolveRepositorySpec(repoArg string) (repository.Repository, error) {
	repoArg = strings.TrimSpace(repoArg)

	if repoArg != "" {
		return repository.Parse(repoArg)
	}

	return repository.Current()
}

func CurrentLogin(host string) (string, error) {
	client, err := newRESTClient(host)
	if err != nil {
		return "", err
	}

	var userResp struct {
		Login string `json:"login"`
	}
	if err := client.Get("user", &userResp); err != nil {
		return "", err
	}
	if userResp.Login == "" {
		return "", fmt.Errorf("failed to resolve current GitHub login")
	}

	return userResp.Login, nil
}
