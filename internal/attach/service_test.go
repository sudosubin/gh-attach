package attach

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceRun_ValidatesCookiesFileFlagsBeforeUploadFile(t *testing.T) {
	t.Parallel()

	_, err := NewService(io.Discard).Run(t.Context(), Request{
		FilePath:    filepath.Join(t.TempDir(), "missing-artifact.zip"),
		Browser:     "edge",
		CookiesFile: "/tmp/github-cookies.json",
	})
	if err == nil {
		t.Fatalf("Run() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "--cookies-file cannot be combined with --browser") {
		t.Fatalf("error = %q", err.Error())
	}
	if strings.Contains(err.Error(), "missing-artifact.zip") {
		t.Fatalf("validated upload file before CLI source conflict: %q", err.Error())
	}
}
