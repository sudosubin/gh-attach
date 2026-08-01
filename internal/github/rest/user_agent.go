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

// ghModulePath is gh's own module path (distinct from BuildInfo.Path, which
// is the *main package* path, e.g. "github.com/cli/cli/v2/cmd/gh"); used to
// reject buildinfo read from a same-named-but-different "gh" binary on PATH.
const ghModulePath = "github.com/cli/cli/v2"

var (
	ghVersionPattern      = regexp.MustCompile(`^gh version ([^\s]+)`)
	ldflagsVersionPattern = regexp.MustCompile(`internal/build\.Version=(\S+)`)
)

// githubCLIAppVersion resolves the installed gh CLI's version. It prefers
// reading the binary's embedded build info (no process spawn) and only
// shells out to "gh version" when that's unavailable (e.g. binaries built
// with -trimpath, which is how Homebrew, nixpkgs, and most distro packages
// build gh).
var githubCLIAppVersion = sync.OnceValue(resolveGHCLIVersion)

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

// ghExecutablePath mirrors go-gh's own lookup order (GH_PATH, then PATH),
// resolved through symlinks since package managers often point PATH at a
// symlink (or, on some nixpkgs setups, a wrapper script) rather than the
// real binary.
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

// versionFromBuildInfoData mirrors gh's own internal/build version
// resolution (ldflags override, else module version), so it stays
// consistent with what "gh version" itself would report.
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
