package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ForceScreenSaver(c *core.X11Conn, mode base.CARD8) error {
	_, err := c.Send(&request.ForceScreenSaverRequest{Mode: mode})
	return err
}
