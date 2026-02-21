package ghapi

import (
	"time"

	goapi "github.com/cli/go-gh/v2/pkg/api"
)

const requestTimeout = 30 * time.Second

func newRESTClient(host string) (*goapi.RESTClient, error) {
	opts := goapi.ClientOptions{Host: host, Timeout: requestTimeout}

	if version := githubCLIAppVersion(); version != "" {
		opts.Headers = map[string]string{"User-Agent": "GitHub CLI " + version}
	}

	return goapi.NewRESTClient(opts)
}
