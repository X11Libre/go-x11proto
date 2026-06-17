package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func DestroyWindow(c *core.X11Conn, window base.WINDOW) error {
	_, err := c.Send(&request.DestroyWindowRequest{Window: window})
	return err
}
