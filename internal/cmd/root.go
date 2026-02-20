package cmd

import "github.com/spf13/cobra"

func NewCmdRoot() *cobra.Command {
	cmd := NewCmdAttach(nil)
	cmd.Use = "gh-attach <file>"
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return cmd
}
