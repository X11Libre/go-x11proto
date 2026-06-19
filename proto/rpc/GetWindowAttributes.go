package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetWindowAttributes(c *core.X11Conn, win base.WINDOW) (*request.GetWindowAttributesReply, error) {
	reply, err := c.SendAndWait(&request.GetWindowAttributesRequest{Window: win})
	if err != nil {
		return nil, err
	}
	rep := &request.GetWindowAttributesReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
