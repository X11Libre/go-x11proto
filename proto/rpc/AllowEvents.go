package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func AllowEvents(c *core.X11Conn, mode base.CARD8, time base.CARD32) error {
	_, err := c.Send(&request.AllowEventsRequest{Mode: mode, Time: time})
	return err
}
