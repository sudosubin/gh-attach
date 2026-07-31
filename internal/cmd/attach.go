package cmd

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sudosubin/gh-attach/internal/app"
	"github.com/sudosubin/gh-attach/internal/cookies"
)

const sessionTokenEnv = "GH_ATTACH_SESSION_TOKEN"

type AttachOptions struct {
	FilePaths       []string
	Repo            string
	Browser         string
	Profile         string
	CookieStorePath string
	SessionToken    string
	Markdown        bool
	JSON            jsonFlags
	Verbose         bool
}

func NewCmdAttach(runF func(*AttachOptions) error) *cobra.Command {
	return newCmdUpload("<file>...", runF)
}

func NewCmdUpload(runF func(*AttachOptions) error) *cobra.Command {
	cmd := newCmdUpload("upload <file>...", runF)
	cmd.Example = strings.ReplaceAll(cmd.Example, "gh attach ", "gh attach upload ")
	return cmd
}

func newCmdUpload(use string, runF func(*AttachOptions) error) *cobra.Command {
	opts := &AttachOptions{}

	cmd := &cobra.Command{
		Use:   use,
		Short: "Upload files to GitHub user-attachments",
		Long:  "Upload files to GitHub user-attachments.",
		Example: `  $ gh attach ./image.png -R owner/repo # Upload to a specific repository
  $ gh attach ./image.png ./report.pdf # Upload multiple files
  $ gh attach ./image.png --browser chrome --profile Default # Use a specific browser and profile for cookies
  $ gh attach ./image.png --markdown # Output a Markdown reference
  $ gh attach ./image.png --json id,href,name # Output specific JSON fields`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.FilePaths = args
			var err error
			opts.SessionToken, err = sessionToken(cmd, opts.SessionToken)
			if err != nil {
				return err
			}
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
	cmd.Flags().StringVar(&opts.SessionToken, "session-token", "", "GitHub user_session cookie value")
	cmd.Flags().BoolVar(&opts.Markdown, "markdown", false, "Output Markdown references")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output")

	addJSONFlags(cmd, &opts.JSON, availableJSONFields())
	cmd.MarkFlagsMutuallyExclusive("markdown", "json")
	for _, name := range []string{"browser", "profile", "cookie-store-path"} {
		cmd.MarkFlagsMutuallyExclusive("session-token", name)
	}

	return cmd
}

func sessionToken(cmd *cobra.Command, flagValue string) (string, error) {
	value, ok := flagValue, cmd.Flags().Changed("session-token")
	if !ok {
		for _, name := range []string{"browser", "profile", "cookie-store-path"} {
			if cmd.Flags().Changed(name) {
				return "", nil
			}
		}
		value, ok = os.LookupEnv(sessionTokenEnv)
	}
	if !ok {
		return "", nil
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("session token is empty")
	}
	return value, nil
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
		SessionToken:    opts.SessionToken,
		Verbose:         opts.Verbose,
	})

	var outputErr error
	if len(assets) > 0 {
		outputErr = writeAssets(os.Stdout, assets, options{
			Markdown:    opts.Markdown,
			JSONFlagSet: opts.JSON.Enabled,
			JSONFields:  opts.JSON.Fields,
			JQ:          opts.JSON.Filter,
			Template:    opts.JSON.Template,
		})
	}

	return errors.Join(uploadErr, outputErr)
}
