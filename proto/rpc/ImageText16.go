package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ImageText16(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, x, y base.INT16, text []base.CARD16) error {
	_, err := c.Send(&request.ImageText16Request{Drawable: drawable, Gc: gc, X: x, Y: y, Text: text})
	return err
}
