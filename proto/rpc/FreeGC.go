package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func FreeGC(c *core.X11Conn, gc base.GC) error {
	_, err := c.Send(&request.FreeGCRequest{Gc: gc})
	return err
}
