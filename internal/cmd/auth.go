package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const sessionTokenEnv = "GH_ATTACH_SESSION_TOKEN"

func sessionToken(cmd *cobra.Command, flagValue string) (string, error) {
	if cmd.Flags().Changed("session-token") {
		value := strings.TrimSpace(flagValue)
		if value == "" {
			return "", errors.New("session token is empty")
		}
		return value, nil
	}
	if browserOptionsChanged(cmd) {
		return "", nil
	}
	return strings.TrimSpace(os.Getenv(sessionTokenEnv)), nil
}

func browserOptionsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("browser") || cmd.Flags().Changed("profile") || cmd.Flags().Changed("cookie-store-path")
}
