package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewCmdUpload_AcceptsFiles(t *testing.T) {
	// given
	var got []string
	cmd := NewCmdUpload(func(opts *AttachOptions) error {
		got = append(got, opts.FilePaths...)
		return nil
	})
	cmd.SetArgs([]string{"image.png", "report.pdf"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// when
	err := cmd.Execute()
	// then
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.Join(got, ",") != "image.png,report.pdf" {
		t.Fatalf("FilePaths = %v", got)
	}
}

func TestNewCmdRoot_DispatchesSubcommands(t *testing.T) {
	// given
	root := NewCmdRoot()
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"image.png"}, want: "gh-attach"},
		{args: []string{"upload", "image.png"}, want: "upload"},
		{args: []string{"download", "https://github.com/user-attachments/assets/id"}, want: "download"},
	}

	for _, test := range tests {
		// when
		cmd, _, err := root.Find(test.args)
		// then
		if err != nil {
			t.Fatalf("Find(%v) error = %v", test.args, err)
		}
		if cmd.Name() != test.want {
			t.Fatalf("Find(%v).Name() = %q, want %q", test.args, cmd.Name(), test.want)
		}
	}
}

func TestNewCmdDownload_AuthenticationPrecedence(t *testing.T) {
	// given
	t.Setenv(sessionTokenEnv, "environment-token")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "environment",
			args: []string{"https://github.com/user-attachments/assets/id", "-O", "image.png"},
			want: "environment-token",
		},
		{
			name: "flag",
			args: []string{
				"https://github.com/user-attachments/assets/id",
				"-O", "image.png",
				"--session-token", "flag-token",
			},
			want: "flag-token",
		},
		{
			name: "explicit browser",
			args: []string{
				"https://github.com/user-attachments/assets/id",
				"-O", "image.png",
				"--browser", "firefox",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			var got string
			cmd := NewCmdDownload(func(opts *DownloadOptions) error {
				got = opts.SessionToken
				return nil
			})
			cmd.SetArgs(test.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			// when
			err := cmd.Execute()
			// then
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("SessionToken = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNewCmdDownload_RequiresOutput(t *testing.T) {
	// given
	cmd := NewCmdDownload(func(*DownloadOptions) error {
		t.Fatal("download ran without --output")
		return nil
	})
	cmd.SetArgs([]string{"https://github.com/user-attachments/assets/id"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// when
	err := cmd.Execute()

	// then
	if err == nil || !strings.Contains(err.Error(), "output") {
		t.Fatalf("Execute() error = %v, want required output error", err)
	}
}

func TestWriteDownload(t *testing.T) {
	t.Run("writes new file", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "asset.txt")

		// when
		err := writeDownload(strings.NewReader("contents"), path, false, io.Discard)
		// then
		if err != nil {
			t.Fatalf("writeDownload() error = %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "contents" {
			t.Fatalf("file = %q", got)
		}
		if info, err := os.Stat(path); runtime.GOOS != "windows" && (err != nil || info.Mode().Perm() != 0o644) {
			t.Fatalf("mode = %v, %v, want 0644", info.Mode(), err)
		}
	})

	t.Run("stdout", func(t *testing.T) {
		// given
		var out bytes.Buffer

		// when
		err := writeDownload(strings.NewReader("contents"), "-", false, &out)
		// then
		if err != nil {
			t.Fatalf("writeDownload() error = %v", err)
		}
		if out.String() != "contents" {
			t.Fatalf("output = %q", out.String())
		}
	})

	t.Run("refuses existing file", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "asset.txt")
		if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
			t.Fatal(err)
		}

		// when
		err := writeDownload(strings.NewReader("replacement"), path, false, io.Discard)

		// then
		if err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("writeDownload() error = %v, want already exists", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "original" {
			t.Fatalf("file = %q, want original", got)
		}
	})

	t.Run("clobbers existing file", func(t *testing.T) {
		// given
		path := filepath.Join(t.TempDir(), "asset.txt")
		if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
			t.Fatal(err)
		}

		// when
		err := writeDownload(strings.NewReader("replacement"), path, true, io.Discard)
		// then
		if err != nil {
			t.Fatalf("writeDownload() error = %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "replacement" {
			t.Fatalf("file = %q, want replacement", got)
		}
	})

	t.Run("removes partial file", func(t *testing.T) {
		// given
		dir := t.TempDir()
		path := filepath.Join(dir, "asset.txt")
		src := io.MultiReader(strings.NewReader("partial"), errorReader{})

		// when
		err := writeDownload(src, path, false, io.Discard)

		// then
		if err == nil {
			t.Fatal("writeDownload() error = nil")
		}
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("output file exists after failure: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("temporary files remain: %v", entries)
		}
	})
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
