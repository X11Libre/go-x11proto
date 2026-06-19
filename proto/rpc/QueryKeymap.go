package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func QueryKeymap(c *core.X11Conn) (*request.QueryKeymapReply, error) {
	reply, err := c.SendAndWait(&request.QueryKeymapRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.QueryKeymapReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
