package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ChangeSaveSet(c *core.X11Conn, mode base.CARD8, win base.WINDOW) error {
	_, err := c.Send(&request.ChangeSaveSetRequest{Mode: mode, Window: win})
	return err
}
