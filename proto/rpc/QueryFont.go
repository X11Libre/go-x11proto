package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func QueryFont(c *core.X11Conn, font base.FONT) (*request.QueryFontReply, error) {
	reply, err := c.SendAndWait(&request.QueryFontRequest{Font: font})
	if err != nil {
		return nil, err
	}
	rep := &request.QueryFontReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
