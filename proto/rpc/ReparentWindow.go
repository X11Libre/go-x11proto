package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ReparentWindow(c *core.X11Conn, win, parent base.WINDOW, x, y base.INT16) error {
	_, err := c.Send(&request.ReparentWindowRequest{Window: win, Parent: parent, X: x, Y: y})
	return err
}
