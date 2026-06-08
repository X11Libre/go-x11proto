package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CreatePixmap(c *core.X11Conn, depth base.CARD8, drawable base.DRAWABLE, width, height base.CARD16) (base.PIXMAP, error) {
	pid := base.PIXMAP(c.NextResourceID())
	req := request.CreatePixmapRequest{
		Depth:    depth,
		Pid:      pid,
		Drawable: drawable,
		Width:    width,
		Height:   height,
	}

	_, err := c.Send(&req)
	return req.Pid, err
}
