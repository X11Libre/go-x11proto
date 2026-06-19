package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetCloseDownMode(c *core.X11Conn, mode base.CARD8) error {
	_, err := c.Send(&request.SetCloseDownModeRequest{Mode: mode})
	return err
}
