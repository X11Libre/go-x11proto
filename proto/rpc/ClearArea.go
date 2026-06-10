package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ClearArea(c *core.X11Conn, window base.WINDOW, x, y base.INT16, width, height base.CARD16, exposures bool) error {
	_, err := c.Send(&request.ClearAreaRequest{
		Window:    window,
		Exposures: exposures,
		X:         x,
		Y:         y,
		Width:     width,
		Height:    height,
	})
	return err
}
