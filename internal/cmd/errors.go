package cmd

import (
	"errors"
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
