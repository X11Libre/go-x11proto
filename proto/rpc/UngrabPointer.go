package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func UngrabPointer(c *core.X11Conn, time base.CARD32) error {
	_, err := c.Send(&request.UngrabPointerRequest{Time: time})
	return err
}
