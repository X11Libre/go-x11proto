package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func StoreNamedColor(c *core.X11Conn, flags base.CARD8, cmap base.COLORMAP, pixel base.CARD32, name string) error {
	_, err := c.Send(&request.StoreNamedColorRequest{Flags: flags, Colormap: cmap, Pixel: pixel, Name: name})
	return err
}
