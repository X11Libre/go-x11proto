package base

import (
	"fmt"
)

type X11Error struct {
	Message string
}

func (e X11Error) Error() string {
	return e.Message
}

func MakeX11Error(msg string) X11Error {
	return X11Error{Message: msg}
}

func MakeX11ErrorF(format string, args ...any) X11Error {
	return X11Error{
		Message: fmt.Sprintf(format, args...),
	}
}
