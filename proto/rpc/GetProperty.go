package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetProperty(c *core.X11Conn, del bool, win base.WINDOW, property, typ base.ATOM, longOffset, longLength base.CARD32) (*request.GetPropertyReply, error) {
	reply, err := c.SendAndWait(&request.GetPropertyRequest{
		Delete: del, Window: win, Property: property, Type: typ,
		LongOffset: longOffset, LongLength: longLength,
	})
	if err != nil {
		return nil, err
	}
	rep := &request.GetPropertyReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
