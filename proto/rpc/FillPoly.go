package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func FillPoly(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, shape, coordMode base.CARD8, points []base.Point) error {
	_, err := c.Send(&request.FillPolyRequest{Drawable: drawable, Gc: gc, Shape: shape, CoordMode: coordMode, Points: points})
	return err
}
