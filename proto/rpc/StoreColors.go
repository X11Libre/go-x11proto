package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func StoreColors(c *core.X11Conn, cmap base.COLORMAP, items []request.ColorItem) error {
	_, err := c.Send(&request.StoreColorsRequest{Colormap: cmap, Items: items})
	return err
}
