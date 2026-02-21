package ghapi

import "fmt"

func LatestCommitSHA(host string, owner string, name string) (string, error) {
	client, err := newRESTClient(host)
	if err != nil {
		return "", err
	}

	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := client.Get(fmt.Sprintf("repos/%s/%s/commits?per_page=1", owner, name), &commits); err != nil {
		return "", err
	}
	if len(commits) == 0 || commits[0].SHA == "" {
		return "", fmt.Errorf("failed to resolve latest commit sha")
	}

	return commits[0].SHA, nil
}
