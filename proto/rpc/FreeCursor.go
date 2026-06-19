package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func FreeCursor(c *core.X11Conn, cursor base.CURSOR) error {
	_, err := c.Send(&request.FreeCursorRequest{Cursor: cursor})
	return err
}
