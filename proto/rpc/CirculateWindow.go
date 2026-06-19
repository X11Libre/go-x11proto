package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CirculateWindow(c *core.X11Conn, direction base.CARD8, win base.WINDOW) error {
	_, err := c.Send(&request.CirculateWindowRequest{Direction: direction, Window: win})
	return err
}
