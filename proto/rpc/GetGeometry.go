package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetGeometry(c *core.X11Conn, drawable base.DRAWABLE) (*request.GetGeometryReply, error) {
	reply, err := c.SendAndWait(&request.GetGeometryRequest{Drawable: drawable})
	if err != nil {
		return nil, err
	}
	rep := &request.GetGeometryReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
