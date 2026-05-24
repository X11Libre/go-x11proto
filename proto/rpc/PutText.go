package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func PutText8(c *core.X11Conn, d base.DRAWABLE, gc base.GC, x base.INT16, y base.INT16, text string) error {
	req := request.PutText8Request{
		Drawable: d,
		Gc:       gc,
		X:        x,
		Y:        y,
		Text:     text,
	}
	_, err := c.Send(&req)
	return err
}
