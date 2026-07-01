package base

import (
	"errors"
	"fmt"
)

type X11Error struct {
	Message string
	wrapped error
}

func (e X11Error) Error() string { return e.Message }

// Unwrap exposes an error wrapped via a %w verb in MakeX11ErrorF, so
// errors.Is / errors.As reach through to the underlying cause.
func (e X11Error) Unwrap() error { return e.wrapped }

func MakeX11Error(msg string) X11Error {
	return X11Error{Message: msg}
}

// MakeX11ErrorF formats like fmt.Errorf: a %w verb renders the wrapped error's
// message into Message and is exposed via Unwrap. (fmt.Sprintf, used before,
// does not understand %w and produced "%!w(...)" while dropping the chain.)
func MakeX11ErrorF(format string, args ...any) X11Error {
	err := fmt.Errorf(format, args...)
	return X11Error{Message: err.Error(), wrapped: errors.Unwrap(err)}
}
