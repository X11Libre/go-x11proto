package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

// CopyColormapAndFree allocates a new colormap id, copies srcMap into it and
// frees srcMap's entries.
func CopyColormapAndFree(c *core.X11Conn, srcMap base.COLORMAP) (base.COLORMAP, error) {
	mid := base.COLORMAP(c.NextResourceID())
	_, err := c.Send(&request.CopyColormapAndFreeRequest{Mid: mid, SrcMap: srcMap})
	return mid, err
}
