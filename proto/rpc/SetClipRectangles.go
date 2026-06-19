package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetClipRectangles(c *core.X11Conn, ordering base.CARD8, gc base.GC, clipX, clipY base.INT16, rects []base.Rectangle) error {
	_, err := c.Send(&request.SetClipRectanglesRequest{
		Ordering: ordering, Gc: gc, ClipXOrigin: clipX, ClipYOrigin: clipY, Rectangles: rects,
	})
	return err
}
