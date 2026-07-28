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
