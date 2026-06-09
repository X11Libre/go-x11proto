package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ChangeWindowAttributes(c *core.X11Conn, req *request.ChangeWindowAttributesRequest) error {
	_, err := c.Send(req)
	return err
}

func SetWindowBackgroundPixmap(c *core.X11Conn, window base.WINDOW, pixmap base.PIXMAP) error {
	req := request.ChangeWindowAttributesRequest{
		Window:     window,
		ValueMask:  request.CW_BACKGROUND_PIXMAP,
		BackPixmap: base.XID(pixmap),
	}
	_, err := c.Send(&req)
	return err
}
