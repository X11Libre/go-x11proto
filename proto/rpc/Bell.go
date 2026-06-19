package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func Bell(c *core.X11Conn, percent int8) error {
	_, err := c.Send(&request.BellRequest{Percent: percent})
	return err
}
