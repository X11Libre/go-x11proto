package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func KillClient(c *core.X11Conn, resource base.CARD32) error {
	_, err := c.Send(&request.KillClientRequest{Resource: resource})
	return err
}
