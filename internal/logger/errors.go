package logger

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrAppNameIsEmpty is returned if Log.AppName was not defined.
	ErrAppNameIsEmpty = errors.New("config Log.AppName can not be empty")

	// ErrServiceNameIsEmpty is returned if Log.ServiceName was not defined.
	ErrServiceNameIsEmpty = errors.New("config Log.ServiceName can not be empty")
)

// ErrorHandler implements a custom error handler.
func ErrorHandler(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "zerolog: could not write event: %v\n", err)
}
