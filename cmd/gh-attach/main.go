package main

import (
	"fmt"
	"os"

	"github.com/sudosubin/gh-attach/internal/cmd"
)

func main() {
	if err := cmd.NewCmdRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Run 'gh-attach --help' for usage.")
		os.Exit(1)
	}
}
