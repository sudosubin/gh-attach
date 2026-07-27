package main

import (
	"fmt"
	"os"

	"github.com/sudosubin/gh-attach/internal/cmd"
)

func main() {
	root := cmd.NewCmdRoot()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if cmd.ShouldShowUsage(err) {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, root.UsageString())
		}
		os.Exit(1)
	}
}
