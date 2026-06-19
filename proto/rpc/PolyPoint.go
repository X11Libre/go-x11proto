package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func PolyPoint(c *core.X11Conn, coordMode base.CARD8, drawable base.DRAWABLE, gc base.GC, points []base.Point) error {
	_, err := c.Send(&request.PolyPointRequest{CoordMode: coordMode, Drawable: drawable, Gc: gc, Points: points})
	return err
}
