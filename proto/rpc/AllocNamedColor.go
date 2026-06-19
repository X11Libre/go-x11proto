package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func AllocNamedColor(c *core.X11Conn, cmap base.COLORMAP, name string) (*request.AllocNamedColorReply, error) {
	reply, err := c.SendAndWait(&request.AllocNamedColorRequest{Colormap: cmap, Name: name})
	if err != nil {
		return nil, err
	}
	rep := &request.AllocNamedColorReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
