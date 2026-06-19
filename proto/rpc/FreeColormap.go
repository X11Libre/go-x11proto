package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func FreeColormap(c *core.X11Conn, cmap base.COLORMAP) error {
	_, err := c.Send(&request.FreeColormapRequest{Colormap: cmap})
	return err
}
