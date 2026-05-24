package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func MapWindow(c *core.X11Conn, win base.WINDOW) error {
	req := request.MapWindowRequest{
		Window: win,
	}

	// request doesn't get a reply
	_, err := c.Send(&req)
	return err
}
