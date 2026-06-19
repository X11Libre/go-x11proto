package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func LookupColor(c *core.X11Conn, cmap base.COLORMAP, name string) (*request.LookupColorReply, error) {
	reply, err := c.SendAndWait(&request.LookupColorRequest{Colormap: cmap, Name: name})
	if err != nil {
		return nil, err
	}
	rep := &request.LookupColorReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
