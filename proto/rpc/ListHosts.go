package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ListHosts(c *core.X11Conn) (*request.ListHostsReply, error) {
	reply, err := c.SendAndWait(&request.ListHostsRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.ListHostsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
