package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func QueryColors(c *core.X11Conn, cmap base.COLORMAP, pixels []base.CARD32) ([]request.Rgb, error) {
	reply, err := c.SendAndWait(&request.QueryColorsRequest{Colormap: cmap, Pixels: pixels})
	if err != nil {
		return nil, err
	}
	rep := &request.QueryColorsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep.Colors, nil
}
