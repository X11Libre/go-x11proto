package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GrabServer(c *core.X11Conn) error {
	_, err := c.Send(&request.GrabServerRequest{})
	return err
}
