package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	cobra.AddTemplateFunc("jsonFields", func(cmd *cobra.Command) string {
		raw := cmd.Annotations["help:json-fields"]
		if raw == "" {
			return ""
		}
		fields := strings.Split(raw, ",")
		return "  " + strings.Join(fields, ", ")
	})
}

func NewCmdRoot() *cobra.Command {
	cmd := NewCmdAttach(nil)
	cmd.Use = "gh-attach <file>..."
	cmd.Short = "Upload and download GitHub user-attachments"
	cmd.Long = "Upload and download GitHub user-attachments."
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetUsageTemplate(usageTemplate)
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.AddCommand(NewCmdUpload(nil), NewCmdDownload(nil))
	return cmd
}

const usageTemplate = `USAGE
  {{.UseLine}}{{if .HasAvailableLocalFlags}}

FLAGS
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

INHERITED FLAGS
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

COMMANDS
{{range .Commands}}{{if .IsAvailableCommand}}  {{rpad .Name .NamePadding }} {{.Short}}
{{end}}{{end}}{{end}}{{if index .Annotations "help:json-fields"}}

JSON FIELDS
{{jsonFields .}}{{end}}{{if .HasExample}}

EXAMPLES
{{.Example | trimTrailingWhitespaces}}{{end}}
`
