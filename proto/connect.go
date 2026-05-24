package proto

import (
	"github.com/X11Libre/go-x11proto/proto/core"
)

func Dial(display_name string) (*core.X11Conn, error) {
	return core.NewConn(display_name, false)
}

func DialBE(display_name string) (*core.X11Conn, error) {
	return core.NewConn(display_name, true)
}
