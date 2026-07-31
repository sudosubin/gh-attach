package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sudosubin/gh-attach/internal/app"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type DownloadOptions struct {
	URL             string
	Output          string
	Clobber         bool
	Browser         string
	Profile         string
	CookieStorePath string
	SessionToken    string
	Verbose         bool
}

func NewCmdDownload(runF func(*DownloadOptions) error) *cobra.Command {
	opts := &DownloadOptions{}

	cmd := &cobra.Command{
		Use:   "download <url>",
		Short: "Download a GitHub user-attachment",
		Long:  "Download a GitHub user-attachment.",
		Example: `  $ gh attach download https://github.com/user-attachments/assets/550e8400-e29b-41d4-a716-446655440000 -O image.png
  $ gh attach download https://github.com/user-attachments/files/123/report.pdf -O -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.URL = args[0]
			var err error
			opts.SessionToken, err = sessionToken(cmd, opts.SessionToken)
			if err != nil {
				return err
			}
			if runF != nil {
				return runF(opts)
			}
			return downloadRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Output, "output", "O", "", "File to write to (use \"-\" for standard output)")
	cmd.Flags().BoolVar(&opts.Clobber, "clobber", false, "Overwrite an existing file")
	cmd.Flags().StringVar(&opts.Browser, "browser", "", "Browser to use ("+cookies.BrowserChoices()+")")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "Browser profile name")
	cmd.Flags().StringVar(&opts.CookieStorePath, "cookie-store-path", "", "Cookie store file path")
	cmd.Flags().StringVar(&opts.SessionToken, "session-token", "", "GitHub user_session cookie value")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output")
	_ = cmd.MarkFlagRequired("output")
	for _, name := range []string{"browser", "profile", "cookie-store-path"} {
		cmd.MarkFlagsMutuallyExclusive("session-token", name)
	}

	return cmd
}

func downloadRun(opts *DownloadOptions) error {
	if err := checkOutput(opts.Output, opts.Clobber); err != nil {
		return err
	}

	body, err := app.NewService(os.Stderr).Download(context.Background(), app.DownloadRequest{
		URL:             opts.URL,
		Browser:         opts.Browser,
		Profile:         opts.Profile,
		CookieStorePath: opts.CookieStorePath,
		SessionToken:    opts.SessionToken,
		Verbose:         opts.Verbose,
	})
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	return writeDownload(body, opts.Output, opts.Clobber, os.Stdout)
}

func errAlreadyExists(output string) error {
	return fmt.Errorf("%s already exists (use `--clobber` to overwrite)", output)
}

func checkOutput(output string, clobber bool) error {
	if output != "-" && !clobber {
		if _, err := os.Stat(output); err == nil {
			return errAlreadyExists(output)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func writeDownload(src io.Reader, output string, clobber bool, stdout io.Writer) error {
	if output == "-" {
		_, err := io.Copy(stdout, src)
		return err
	}
	if err := checkOutput(output, clobber); err != nil {
		return err
	}

	dir := filepath.Dir(output)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(output)+".*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// os.CreateTemp uses 0600; downloads are ordinary files, 0644 like gh's.
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if !clobber {
		// O_EXCL claims the name atomically; the rename below replaces only our reservation.
		reserved, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			return errAlreadyExists(output)
		}
		if err != nil {
			return err
		}
		_ = reserved.Close()
	}
	return os.Rename(tmpPath, output)
}
