package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func FillRect(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, x base.INT16, y base.INT16, w base.CARD16, h base.CARD16) error {
	return FillRects(
		c,
		drawable,
		gc,
		[]base.Rectangle{
			{X: x, Y: y, Width: w, Height: h},
		},
	)
}

func FillRects(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, rects []base.Rectangle) error {
	req := request.PolyFillRectRequest{
		Drawable: drawable,
		Gc:       gc,
		Rects:    rects,
	}

	_, err := c.Send(&req)
	return err
}
