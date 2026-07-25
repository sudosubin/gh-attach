package rest

import (
	"regexp"
	"strings"
	"sync"

	gh "github.com/cli/go-gh/v2"
)

var (
	ghVersionPattern = regexp.MustCompile(`^gh version ([^\s]+)`)
	appVersionOnce   sync.Once
	appVersionValue  string
)

func githubCLIAppVersion() string {
	appVersionOnce.Do(func() {
		stdout, _, err := gh.Exec("version")
		if err != nil {
			return
		}

		appVersionValue = parseGHCLIVersion(stdout.String())
	})

	return appVersionValue
}

func parseGHCLIVersion(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := ghVersionPattern.FindStringSubmatch(line)
		if len(m) > 1 {
			return m[1]
		}
	}

	return ""
}
