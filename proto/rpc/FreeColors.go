package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func FreeColors(c *core.X11Conn, cmap base.COLORMAP, planeMask base.CARD32, pixels []base.CARD32) error {
	_, err := c.Send(&request.FreeColorsRequest{Colormap: cmap, PlaneMask: planeMask, Pixels: pixels})
	return err
}
