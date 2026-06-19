package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func PolyFillArc(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, arcs []base.Arc) error {
	_, err := c.Send(&request.PolyFillArcRequest{Drawable: drawable, Gc: gc, Arcs: arcs})
	return err
}
