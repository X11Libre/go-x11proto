package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func DeleteProperty(c *core.X11Conn, win base.WINDOW, property base.ATOM) error {
	_, err := c.Send(&request.DeletePropertyRequest{Window: win, Property: property})
	return err
}
