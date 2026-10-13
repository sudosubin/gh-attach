package rest

import "fmt"

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
	if repoResp.ID <= 0 {
		return Repository{}, fmt.Errorf("resolved repository %s/%s has no valid id", owner, name)
	}

	return Repository{
		Host:  s.host,
		Owner: owner,
		Name:  name,
		ID:    repoResp.ID,
	}, nil
}
