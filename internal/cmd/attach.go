package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/sudosubin/gh-attach/internal/attach"
	"github.com/sudosubin/gh-attach/internal/cmdutil"
	"github.com/sudosubin/gh-attach/internal/cookies"
	"github.com/sudosubin/gh-attach/internal/formatting"
)

type AttachOptions struct {
	FilePath        string
	Repo            string
	Browser         string
	Profile         string
	CookieStorePath string
	JSON            cmdutil.JSONFlags
	Verbose         bool
}

func NewCmdAttach(runF func(*AttachOptions) error) *cobra.Command {
	opts := &AttachOptions{}

	cmd := &cobra.Command{
		Use:   "<file>",
		Short: "Upload a file to GitHub user-attachments",
		Long:  "Upload a file to GitHub user-attachments.",
		Example: `  $ gh attach ./image.png -R owner/repo # Upload to a specific repository
  $ gh attach ./image.png --browser chrome --profile Default # Use a specific browser and profile for cookies
  $ gh attach ./image.png --json id,href,name # Output specific JSON fields`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.FilePath = args[0]
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

	cmdutil.AddJSONFlags(cmd, &opts.JSON, formatting.AvailableJSONFields())

	return cmd
}

func attachRun(opts *AttachOptions) error {
	ctx := context.Background()

	svc := attach.NewService(os.Stderr)
	asset, err := svc.Run(ctx, attach.Request{
		FilePath:        opts.FilePath,
		Repo:            opts.Repo,
		Browser:         opts.Browser,
		Profile:         opts.Profile,
		CookieStorePath: opts.CookieStorePath,
		Verbose:         opts.Verbose,
	})
	if err != nil {
		return err
	}

	return formatting.WriteAsset(os.Stdout, asset, formatting.Options{
		JSONFlagSet: opts.JSON.Enabled,
		JSONFields:  opts.JSON.Fields,
		JQ:          opts.JSON.Filter,
		Template:    opts.JSON.Template,
	})
}
