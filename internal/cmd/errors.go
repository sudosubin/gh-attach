package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

type flagError struct {
	err error
}

func (e *flagError) Error() string {
	return e.err.Error()
}

func (e *flagError) Unwrap() error {
	return e.err
}

func ShouldShowUsage(err error) bool {
	_, ok := errors.AsType[*flagError](err)
	return ok
}

func setFlagErrorHandler(cmd *cobra.Command) {
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if cmd.HasParent() {
			return cmd.Parent().FlagErrorFunc()(c, err)
		}
		return &flagError{err: err}
	})
}
