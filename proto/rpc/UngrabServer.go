package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func UngrabServer(c *core.X11Conn) error {
	_, err := c.Send(&request.UngrabServerRequest{})
	return err
}
