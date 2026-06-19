package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// CreateColormap allocates a colormap id and creates it.
func CreateColormap(c *core.X11Conn, alloc base.CARD8, window base.WINDOW, visual base.VISUAL) (base.COLORMAP, error) {
	mid := base.COLORMAP(c.NextResourceID())
	_, err := c.Send(&request.CreateColormapRequest{Alloc: alloc, Mid: mid, Window: window, Visual: visual})
	return mid, err
}
