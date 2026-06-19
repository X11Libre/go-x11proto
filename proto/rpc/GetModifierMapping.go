package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetModifierMapping(c *core.X11Conn) (*request.GetModifierMappingReply, error) {
	reply, err := c.SendAndWait(&request.GetModifierMappingRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.GetModifierMappingReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
