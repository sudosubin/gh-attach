package ghapi

import (
	"fmt"
	"strings"
	"time"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
	gauth "github.com/cli/go-gh/v2/pkg/auth"
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

func ResolveRepository(repoArg string, hostnameArg string) (Repository, error) {
	parsed, err := resolveRepositorySpec(repoArg, hostnameArg, CurrentLogin)
	if err != nil {
		return Repository{}, err
	}

	client, err := ghapi.NewRESTClient(ghapi.ClientOptions{Host: parsed.Host, Timeout: 30 * time.Second})
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

func resolveRepositorySpec(
	repoArg string,
	hostnameArg string,
	currentLogin func(host string) (string, error),
) (repository.Repository, error) {
	repoArg = strings.TrimSpace(repoArg)
	hostnameArg = strings.TrimSpace(hostnameArg)

	if repoArg != "" {
		repoSelector := repoArg
		if !strings.Contains(repoSelector, "/") {
			host := hostnameArg
			if host == "" {
				host, _ = gauth.DefaultHost()
			}

			currentUser, err := currentLogin(host)
			if err != nil {
				return repository.Repository{}, fmt.Errorf("resolve current login: %w", err)
			}
			repoSelector = currentUser + "/" + repoSelector
		}

		if hostnameArg != "" {
			parsed, err := repository.ParseWithHost(repoSelector, hostnameArg)
			if err != nil {
				return repository.Repository{}, err
			}
			parsed.Host = hostnameArg
			return parsed, nil
		}

		parsed, err := repository.Parse(repoSelector)
		if err != nil {
			return repository.Repository{}, err
		}
		return parsed, nil
	}

	parsed, err := repository.Current()
	if err != nil {
		return repository.Repository{}, err
	}

	if hostnameArg != "" {
		parsed.Host = hostnameArg
	}

	return parsed, nil
}

func CurrentLogin(host string) (string, error) {
	client, err := ghapi.NewRESTClient(ghapi.ClientOptions{Host: host, Timeout: 30 * time.Second})
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
