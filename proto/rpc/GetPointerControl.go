package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetPointerControl(c *core.X11Conn) (*request.GetPointerControlReply, error) {
	reply, err := c.SendAndWait(&request.GetPointerControlRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.GetPointerControlReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
