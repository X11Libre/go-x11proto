package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetKeyboardControl(c *core.X11Conn) (*request.GetKeyboardControlReply, error) {
	reply, err := c.SendAndWait(&request.GetKeyboardControlRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.GetKeyboardControlReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
