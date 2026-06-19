package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetPointerMapping(c *core.X11Conn) ([]byte, error) {
	reply, err := c.SendAndWait(&request.GetPointerMappingRequest{})
	if err != nil {
		return nil, err
	}
	rep := &request.GetPointerMappingReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep.Map, nil
}
