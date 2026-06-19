package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetInputFocus(c *core.X11Conn, revertTo base.CARD8, focus base.WINDOW, time base.CARD32) error {
	_, err := c.Send(&request.SetInputFocusRequest{RevertTo: revertTo, Focus: focus, Time: time})
	return err
}
