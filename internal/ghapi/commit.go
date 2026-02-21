package ghapi

import "fmt"

func (s *Service) LatestCommitSHA(owner string, name string) (string, error) {
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := s.client.Get(fmt.Sprintf("repos/%s/%s/commits?per_page=1", owner, name), &commits); err != nil {
		return "", err
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", fmt.Errorf("failed to resolve latest commit sha")
	}

	return commits[0].SHA, nil
}
