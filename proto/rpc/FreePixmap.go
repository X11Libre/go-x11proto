package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func FreePixmap(c *core.X11Conn, pixmap base.PIXMAP) error {
	_, err := c.Send(&request.FreePixmapRequest{Pixmap: pixmap})
	return err
}
