package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func TranslateCoordinates(c *core.X11Conn, srcWin, dstWin base.WINDOW, srcX, srcY base.INT16) (*request.TranslateCoordinatesReply, error) {
	reply, err := c.SendAndWait(&request.TranslateCoordinatesRequest{SrcWindow: srcWin, DstWindow: dstWin, SrcX: srcX, SrcY: srcY})
	if err != nil {
		return nil, err
	}
	rep := &request.TranslateCoordinatesReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
