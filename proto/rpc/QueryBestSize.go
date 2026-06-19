package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func QueryBestSize(c *core.X11Conn, class base.CARD8, drawable base.DRAWABLE, width, height base.CARD16) (*request.QueryBestSizeReply, error) {
	reply, err := c.SendAndWait(&request.QueryBestSizeRequest{Class: class, Drawable: drawable, Width: width, Height: height})
	if err != nil {
		return nil, err
	}
	rep := &request.QueryBestSizeReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
