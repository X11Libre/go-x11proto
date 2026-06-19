package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func ListInstalledColormaps(c *core.X11Conn, window base.WINDOW) ([]base.COLORMAP, error) {
	reply, err := c.SendAndWait(&request.ListInstalledColormapsRequest{Window: window})
	if err != nil {
		return nil, err
	}
	rep := &request.ListInstalledColormapsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep.Colormaps, nil
}
