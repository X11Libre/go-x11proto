package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ConfigureWindow(c *core.X11Conn, req *request.ConfigureWindowRequest) error {
	_, err := c.Send(req)
	return err
}
