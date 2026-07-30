package cmd

import (
	"io"
	"slices"
	"strings"
	"testing"
)

func TestNewCmdAttach_AcceptsMultipleFiles(t *testing.T) {
	var got []string
	cmd := NewCmdAttach(func(opts *AttachOptions) error {
		got = append(got, opts.FilePaths...)
		return nil
	})
	cmd.SetArgs([]string{"image.png", "report.pdf"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if want := []string{"image.png", "report.pdf"}; !slices.Equal(got, want) {
		t.Fatalf("FilePaths = %v, want %v", got, want)
	}
}

func TestNewCmdAttach_Markdown(t *testing.T) {
	var markdown bool
	cmd := NewCmdAttach(func(opts *AttachOptions) error {
		markdown = opts.Markdown
		return nil
	})
	cmd.SetArgs([]string{"image.png", "--markdown"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !markdown {
		t.Fatal("Markdown = false, want true")
	}
}

func TestNewCmdAttach_RejectsMarkdownWithJSON(t *testing.T) {
	called := false
	cmd := NewCmdAttach(func(_ *AttachOptions) error {
		called = true
		return nil
	})
	cmd.SetArgs([]string{"image.png", "--markdown", "--json", "href"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "markdown json") {
		t.Fatalf("Execute() error = %v, want mutually exclusive flags error", err)
	}
	if called {
		t.Fatal("upload ran despite incompatible output flags")
	}
}

func TestNewCmdAttach_SessionTokenPrecedence(t *testing.T) {
	t.Setenv(sessionTokenEnv, "environment-token")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "environment", args: []string{"image.png"}, want: "environment-token"},
		{name: "flag", args: []string{"image.png", "--session-token", "flag-token"}, want: "flag-token"},
		{name: "explicit browser", args: []string{"image.png", "--browser", "firefox"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got string
			cmd := NewCmdAttach(func(opts *AttachOptions) error {
				got = opts.SessionToken
				return nil
			})
			cmd.SetArgs(test.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("SessionToken = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNewCmdAttach_RejectsSessionTokenWithBrowserOptions(t *testing.T) {
	for _, flag := range []string{"--browser", "--profile", "--cookie-store-path"} {
		t.Run(flag, func(t *testing.T) {
			cmd := NewCmdAttach(nil)
			cmd.SetArgs([]string{"image.png", "--session-token", "secret", flag, "value"})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "session-token") {
				t.Fatalf("Execute() error = %v, want mutually exclusive flags error", err)
			}
		})
	}
}

func TestNewCmdAttach_AllowsBrowserWithProfile(t *testing.T) {
	cmd := NewCmdAttach(func(_ *AttachOptions) error { return nil })
	cmd.SetArgs([]string{"image.png", "--browser", "firefox", "--profile", "default"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestNewCmdAttach_RejectsEmptySessionToken(t *testing.T) {
	called := false
	cmd := NewCmdAttach(func(_ *AttachOptions) error {
		called = true
		return nil
	})
	cmd.SetArgs([]string{"image.png", "--session-token="})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "session token is empty") {
		t.Fatalf("Execute() error = %v, want empty token error", err)
	}
	if called {
		t.Fatal("upload ran with an empty session token")
	}
}

func TestNewCmdAttach_RejectsEmptySessionTokenEnvironment(t *testing.T) {
	t.Setenv(sessionTokenEnv, " ")
	cmd := NewCmdAttach(func(_ *AttachOptions) error {
		t.Fatal("upload ran with an empty session token")
		return nil
	})
	cmd.SetArgs([]string{"image.png"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "session token is empty") {
		t.Fatalf("Execute() error = %v, want empty token error", err)
	}
}
