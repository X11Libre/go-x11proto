package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func RotateProperties(c *core.X11Conn, window base.WINDOW, delta base.INT16, properties []base.ATOM) error {
	_, err := c.Send(&request.RotatePropertiesRequest{Window: window, Delta: delta, Properties: properties})
	return err
}
