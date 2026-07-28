package app

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/sudosubin/gh-attach/internal/github/attachments"
)

func TestUploadFiles_ContinuesAfterError(t *testing.T) {
	var called []string
	assets, err := uploadFiles([]string{"one.png", "bad.png", "two.png"}, func(path string) (attachments.Asset, error) {
		called = append(called, path)
		if path == "bad.png" {
			return attachments.Asset{}, errors.New("rejected")
		}
		return attachments.Asset{Href: "https://example.com/" + path}, nil
	})

	if want := []string{"one.png", "bad.png", "two.png"}; !slices.Equal(called, want) {
		t.Fatalf("called = %v, want %v", called, want)
	}
	if len(assets) != 2 {
		t.Fatalf("len(assets) = %d, want 2", len(assets))
	}
	if err == nil || !strings.Contains(err.Error(), "bad.png: rejected") {
		t.Fatalf("error = %v, want file-scoped rejection", err)
	}
}
