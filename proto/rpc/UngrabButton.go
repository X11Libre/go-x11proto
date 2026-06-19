package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func UngrabButton(c *core.X11Conn, button base.CARD8, grabWindow base.WINDOW, modifiers base.CARD16) error {
	_, err := c.Send(&request.UngrabButtonRequest{Button: button, GrabWindow: grabWindow, Modifiers: modifiers})
	return err
}
