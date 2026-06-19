package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func SetPointerMapping(c *core.X11Conn, m []base.CARD8) (base.CARD8, error) {
	reply, err := c.SendAndWait(&request.SetPointerMappingRequest{Map: m})
	if err != nil {
		return 0, err
	}
	rep := &request.SetPointerMappingReply{}
	if err := rep.Parse(*reply); err != nil {
		return 0, err
	}
	return rep.Status, nil
}
