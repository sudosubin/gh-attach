package app

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sudosubin/gh-attach/internal/github/attachments"
)

func TestUploadFiles_ContinuesAfterError(t *testing.T) {
	assets, err := uploadFiles([]string{"one.png", "bad.png", "two.png"}, func(path string) (attachments.Asset, error) {
		if path == "bad.png" {
			return attachments.Asset{}, errors.New("rejected")
		}
		return attachments.Asset{Href: "https://example.com/" + path}, nil
	})

	if len(assets) != 2 {
		t.Fatalf("len(assets) = %d, want 2", len(assets))
	}
	got := []string{assets[0].Href, assets[1].Href}
	want := []string{"https://example.com/one.png", "https://example.com/two.png"}
	if !slices.Equal(got, want) {
		t.Fatalf("assets = %v, want %v", got, want)
	}
	if err == nil || !strings.Contains(err.Error(), "bad.png: rejected") {
		t.Fatalf("error = %v, want file-scoped rejection", err)
	}
}

func TestUploadFiles_UploadsConcurrently(t *testing.T) {
	started := make(chan struct{}, maxConcurrentUploads)
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = uploadFiles([]string{"one.png", "two.png"}, func(string) (attachments.Asset, error) {
			started <- struct{}{}
			<-release
			return attachments.Asset{}, nil
		})
	}()

	for range maxConcurrentUploads {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(release)
			t.Fatal("uploads did not start concurrently")
		}
	}
	close(release)
	<-done
}
