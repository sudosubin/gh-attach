package cmd

import (
	"io"
	"slices"
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
