package rest

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func TestParseGHCLIVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name: "standard gh version output",
			output: "gh version 2.86.0 (2026-01-21)\n" +
				"https://github.com/cli/cli/releases/tag/v2.86.0\n",
			want: "2.86.0",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
		{
			name:   "non matching output",
			output: "some unexpected output",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseGHCLIVersion(tt.output); got != tt.want {
				t.Fatalf("parseGHCLIVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionFromBuildInfoData(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "official goreleaser binary: stamped module version",
			info: &debug.BuildInfo{
				Path: "github.com/cli/cli/v2/cmd/gh",
				Main: debug.Module{Path: ghModulePath, Version: "v2.96.0"},
				Settings: []debug.BuildSetting{
					{Key: "-ldflags", Value: "-s -w -X github.com/cli/cli/v2/internal/build.Version=2.96.0"},
				},
			},
			want: "2.96.0",
		},
		{
			name: "fedora rpm: devel module, ldflags carries the version",
			info: &debug.BuildInfo{
				Path: "github.com/cli/cli/v2/cmd/gh",
				Main: debug.Module{Path: ghModulePath, Version: "(devel)"},
				Settings: []debug.BuildSetting{
					{Key: "-ldflags", Value: "-X github.com/cli/cli/v2/internal/build.Version=2.97.0 -linkmode=external"},
				},
			},
			want: "2.97.0",
		},
		{
			name: "homebrew/nixpkgs: -trimpath build, no version signal at all",
			info: &debug.BuildInfo{
				Path:     "github.com/cli/cli/v2/cmd/gh",
				Main:     debug.Module{Path: ghModulePath, Version: "(devel)"},
				Settings: []debug.BuildSetting{{Key: "-trimpath", Value: "true"}},
			},
			want: "",
		},
		{
			name: "go install @version: module version, no ldflags",
			info: &debug.BuildInfo{
				Path: "github.com/cli/cli/v2/cmd/gh",
				Main: debug.Module{Path: ghModulePath, Version: "v2.90.1"},
			},
			want: "2.90.1",
		},
		{
			name: "different program named gh on PATH: module path mismatch",
			info: &debug.BuildInfo{
				Path: "github.com/someone/gh/cmd/gh",
				Main: debug.Module{Path: "github.com/someone/gh", Version: "v9.9.9"},
			},
			want: "",
		},
		{
			name: "empty module version and no ldflags entry",
			info: &debug.BuildInfo{
				Path:     "github.com/cli/cli/v2/cmd/gh",
				Main:     debug.Module{Path: ghModulePath},
				Settings: []debug.BuildSetting{{Key: "-compiler", Value: "gc"}},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := versionFromBuildInfoData(tt.info); got != tt.want {
				t.Fatalf("versionFromBuildInfoData() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionFromBuildInfo_NotAGoBinary(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "gh")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho fake\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	if got := versionFromBuildInfo(fake); got != "" {
		t.Fatalf("versionFromBuildInfo() = %q, want empty for a non-Go-binary path", got)
	}
}

func TestVersionFromBuildInfo_EmptyPath(t *testing.T) {
	if got := versionFromBuildInfo(""); got != "" {
		t.Fatalf("versionFromBuildInfo(\"\") = %q, want empty", got)
	}
}

func TestGHExecutablePath_HonorsGHPath(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "gh")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	t.Setenv("GH_PATH", fake)

	got := ghExecutablePath()
	want, err := filepath.EvalSymlinks(fake)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", fake, err)
	}
	if got != want {
		t.Fatalf("ghExecutablePath() = %q, want %q", got, want)
	}
}
