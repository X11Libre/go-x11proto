package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func AllocColor(c *core.X11Conn, cmap base.COLORMAP, red, green, blue base.CARD16) (*request.AllocColorReply, error) {
	reply, err := c.SendAndWait(&request.AllocColorRequest{Colormap: cmap, Red: red, Green: green, Blue: blue})
	if err != nil {
		return nil, err
	}
	rep := &request.AllocColorReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
