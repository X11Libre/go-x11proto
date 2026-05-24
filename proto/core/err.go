package core

import (
	"github.com/X11Libre/go-x11proto/proto/base"
)

type X11ConnError struct {
	base.X11Error
	Conn *X11Conn
}

func MakeX11ConnErrorF(format string, args ...any) error {
	return X11ConnError{
		X11Error: base.MakeX11ErrorF("X11ConnError: "+format, args...),
	}
}
