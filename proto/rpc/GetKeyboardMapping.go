package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetKeyboardMapping(c *core.X11Conn, firstKeycode, count base.CARD8) (*request.GetKeyboardMappingReply, error) {
	reply, err := c.SendAndWait(&request.GetKeyboardMappingRequest{FirstKeycode: firstKeycode, Count: count})
	if err != nil {
		return nil, err
	}
	rep := &request.GetKeyboardMappingReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
