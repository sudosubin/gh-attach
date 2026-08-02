package rest

import (
	"debug/buildinfo"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"

	gh "github.com/cli/go-gh/v2"
	"github.com/cli/safeexec"
)

// ghModulePath is gh's module path, not BuildInfo.Path (its main package).
const ghModulePath = "github.com/cli/cli/v2"

var (
	ghVersionPattern      = regexp.MustCompile(`^gh version ([^\s]+)`)
	ldflagsVersionPattern = regexp.MustCompile(`internal/build\.Version=(\S+)`)
)

var githubCLIAppVersion = sync.OnceValue(resolveGHCLIVersion)

// resolveGHCLIVersion reads build info; exec fallback for -trimpath builds.
func resolveGHCLIVersion() string {
	if v := versionFromBuildInfo(ghExecutablePath()); v != "" {
		return v
	}

	stdout, _, err := gh.Exec("version")
	if err != nil {
		return ""
	}
	return parseGHCLIVersion(stdout.String())
}

// ghExecutablePath: go-gh's lookup (GH_PATH then PATH), symlinks resolved.
func ghExecutablePath() string {
	exe := os.Getenv("GH_PATH")
	if exe == "" {
		exe, _ = safeexec.LookPath("gh")
	}
	if exe == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

func versionFromBuildInfo(exe string) string {
	if exe == "" {
		return ""
	}
	info, err := buildinfo.ReadFile(exe)
	if err != nil {
		return ""
	}
	return versionFromBuildInfoData(info)
}

// versionFromBuildInfoData mirrors gh's own internal/build version resolution.
func versionFromBuildInfoData(info *debug.BuildInfo) string {
	if info.Main.Path != ghModulePath {
		return ""
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return strings.TrimPrefix(v, "v")
	}
	for _, s := range info.Settings {
		if s.Key != "-ldflags" {
			continue
		}
		if m := ldflagsVersionPattern.FindStringSubmatch(s.Value); len(m) > 1 {
			return m[1]
		}
	}
	return ""
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
