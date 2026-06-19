package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ChangeActivePointerGrab(c *core.X11Conn, cursor base.CURSOR, time base.CARD32, eventMask base.CARD16) error {
	_, err := c.Send(&request.ChangeActivePointerGrabRequest{Cursor: cursor, Time: time, EventMask: eventMask})
	return err
}
