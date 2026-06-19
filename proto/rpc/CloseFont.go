package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CloseFont(c *core.X11Conn, font base.FONT) error {
	_, err := c.Send(&request.CloseFontRequest{Font: font})
	return err
}
