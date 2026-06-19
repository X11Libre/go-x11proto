package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetMotionEvents(c *core.X11Conn, win base.WINDOW, start, stop base.CARD32) (*request.GetMotionEventsReply, error) {
	reply, err := c.SendAndWait(&request.GetMotionEventsRequest{Window: win, Start: start, Stop: stop})
	if err != nil {
		return nil, err
	}
	rep := &request.GetMotionEventsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
