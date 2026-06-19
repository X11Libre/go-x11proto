package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func PolyRectangle(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, rects []base.Rectangle) error {
	_, err := c.Send(&request.PolyRectangleRequest{Drawable: drawable, Gc: gc, Rectangles: rects})
	return err
}
