package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func UnmapSubwindows(c *core.X11Conn, win base.WINDOW) error {
	_, err := c.Send(&request.UnmapSubwindowsRequest{Window: win})
	return err
}
