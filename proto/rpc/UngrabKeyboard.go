package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func UngrabKeyboard(c *core.X11Conn, time base.CARD32) error {
	_, err := c.Send(&request.UngrabKeyboardRequest{Time: time})
	return err
}
