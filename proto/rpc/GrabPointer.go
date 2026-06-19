package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GrabPointer(c *core.X11Conn, req *request.GrabPointerRequest) (*request.GrabPointerReply, error) {
	reply, err := c.SendAndWait(req)
	if err != nil {
		return nil, err
	}
	rep := &request.GrabPointerReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
