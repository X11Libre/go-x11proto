package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func AllocColorCells(c *core.X11Conn, contiguous bool, cmap base.COLORMAP, colors, planes base.CARD16) (*request.AllocColorCellsReply, error) {
	reply, err := c.SendAndWait(&request.AllocColorCellsRequest{Contiguous: contiguous, Colormap: cmap, Colors: colors, Planes: planes})
	if err != nil {
		return nil, err
	}
	rep := &request.AllocColorCellsReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
