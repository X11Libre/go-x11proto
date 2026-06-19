package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GrabButton(c *core.X11Conn, req *request.GrabButtonRequest) error {
	_, err := c.Send(req)
	return err
}
