package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func UngrabKey(c *core.X11Conn, key base.CARD8, grabWindow base.WINDOW, modifiers base.CARD16) error {
	_, err := c.Send(&request.UngrabKeyRequest{Key: key, GrabWindow: grabWindow, Modifiers: modifiers})
	return err
}
