package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func AllocColorPlanes(c *core.X11Conn, contiguous bool, cmap base.COLORMAP, colors, reds, greens, blues base.CARD16) (*request.AllocColorPlanesReply, error) {
	reply, err := c.SendAndWait(&request.AllocColorPlanesRequest{
		Contiguous: contiguous, Colormap: cmap, Colors: colors, Reds: reds, Greens: greens, Blues: blues,
	})
	if err != nil {
		return nil, err
	}
	rep := &request.AllocColorPlanesReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
