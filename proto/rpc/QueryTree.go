package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func QueryTree(c *core.X11Conn, win base.WINDOW) (*request.QueryTreeReply, error) {
	reply, err := c.SendAndWait(&request.QueryTreeRequest{Window: win})
	if err != nil {
		return nil, err
	}
	rep := &request.QueryTreeReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
