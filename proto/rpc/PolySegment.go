package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func PolySegment(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, segments []base.Segment) error {
	_, err := c.Send(&request.PolySegmentRequest{Drawable: drawable, Gc: gc, Segments: segments})
	return err
}
