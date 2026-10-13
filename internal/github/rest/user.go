package rest

import "fmt"

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
