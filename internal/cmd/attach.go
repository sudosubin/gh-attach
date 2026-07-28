package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/sudosubin/gh-attach/internal/app"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

type AttachOptions struct {
	FilePaths       []string
	Repo            string
	Browser         string
	Profile         string
	CookieStorePath string
	JSON            jsonFlags
	Verbose         bool
}

func NewCmdAttach(runF func(*AttachOptions) error) *cobra.Command {
	opts := &AttachOptions{}

	cmd := &cobra.Command{
		Use:   "<file>...",
		Short: "Upload files to GitHub user-attachments",
		Long:  "Upload files to GitHub user-attachments.",
		Example: `  $ gh attach ./image.png -R owner/repo # Upload to a specific repository
  $ gh attach ./image.png ./report.pdf # Upload multiple files
  $ gh attach ./image.png --browser chrome --profile Default # Use a specific browser and profile for cookies
  $ gh attach ./image.png --json id,href,name # Output specific JSON fields`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			opts.FilePaths = args
			if runF != nil {
				return runF(opts)
			}
			return attachRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "[HOST/]OWNER/REPO")
	cmd.Flags().StringVar(&opts.Browser, "browser", "", "Browser to use ("+cookies.BrowserChoices()+")")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "Browser profile name")
	cmd.Flags().StringVar(&opts.CookieStorePath, "cookie-store-path", "", "Cookie store file path")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output")

	addJSONFlags(cmd, &opts.JSON, availableJSONFields())

	return cmd
}

func attachRun(opts *AttachOptions) error {
	ctx := context.Background()

	svc := app.NewService(os.Stderr)
	assets, uploadErr := svc.Run(ctx, app.Request{
		FilePaths:       opts.FilePaths,
		Repo:            opts.Repo,
		Browser:         opts.Browser,
		Profile:         opts.Profile,
		CookieStorePath: opts.CookieStorePath,
		Verbose:         opts.Verbose,
	})

	var outputErr error
	if len(assets) > 0 {
		outputErr = writeAssets(os.Stdout, assets, options{
			JSONFlagSet: opts.JSON.Enabled,
			JSONFields:  opts.JSON.Fields,
			JQ:          opts.JSON.Filter,
			Template:    opts.JSON.Template,
		})
	}

	return errors.Join(uploadErr, outputErr)
}
