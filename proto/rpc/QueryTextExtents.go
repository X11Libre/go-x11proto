package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func QueryTextExtents(c *core.X11Conn, font base.FONT, text []base.CARD16) (*request.QueryTextExtentsReply, error) {
	reply, err := c.SendAndWait(&request.QueryTextExtentsRequest{Font: font, Text: text})
	if err != nil {
		return nil, err
	}
	rep := &request.QueryTextExtentsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
