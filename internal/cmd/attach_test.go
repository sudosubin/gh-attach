package cmd

import "testing"

func TestNewCmdAttach_CapturesCookiesFile(t *testing.T) {
	t.Parallel()

	var got AttachOptions
	command := NewCmdAttach(func(opts *AttachOptions) error {
		got = *opts
		return nil
	})
	command.SetArgs([]string{
		"artifact.zip",
		"--cookies-file",
		"/tmp/github-cookies.json",
	})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.FilePath != "artifact.zip" {
		t.Fatalf("FilePath = %q, want %q", got.FilePath, "artifact.zip")
	}
	if got.CookiesFile != "/tmp/github-cookies.json" {
		t.Fatalf("CookiesFile = %q, want %q", got.CookiesFile, "/tmp/github-cookies.json")
	}
}
