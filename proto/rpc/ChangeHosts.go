package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ChangeHosts(c *core.X11Conn, mode, family base.CARD8, address []byte) error {
	_, err := c.Send(&request.ChangeHostsRequest{Mode: mode, Family: family, Address: address})
	return err
}
