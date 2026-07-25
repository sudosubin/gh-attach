package rest

import (
	"fmt"
	"strings"
	"time"

	goapi "github.com/cli/go-gh/v2/pkg/api"
)

const requestTimeout = 30 * time.Second

type RESTGetter interface {
	Get(path string, data any) error
}

type Service struct {
	host   string
	client RESTGetter
}

func NewService(host string, client RESTGetter) (*Service, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	if client == nil {
		restClient, err := newRESTClient(host)
		if err != nil {
			return nil, err
		}
		client = restClient
	}

	return &Service{host: host, client: client}, nil
}

func newRESTClient(host string) (*goapi.RESTClient, error) {
	opts := goapi.ClientOptions{Host: host, Timeout: requestTimeout}

	if version := githubCLIAppVersion(); version != "" {
		opts.Headers = map[string]string{"User-Agent": "GitHub CLI " + version}
	}

	return goapi.NewRESTClient(opts)
}
