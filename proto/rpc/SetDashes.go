package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetDashes(c *core.X11Conn, gc base.GC, dashOffset base.CARD16, dashes []base.CARD8) error {
	_, err := c.Send(&request.SetDashesRequest{Gc: gc, DashOffset: dashOffset, Dashes: dashes})
	return err
}
