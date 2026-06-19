package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func GetImage(c *core.X11Conn, format base.CARD8, drawable base.DRAWABLE, x, y base.INT16, width, height base.CARD16, planeMask base.CARD32) (*request.GetImageReply, error) {
	reply, err := c.SendAndWait(&request.GetImageRequest{
		Format: format, Drawable: drawable, X: x, Y: y, Width: width, Height: height, PlaneMask: planeMask,
	})
	if err != nil {
		return nil, err
	}
	rep := &request.GetImageReply{}
	if err := rep.Parse(*reply); err != nil {
		return nil, err
	}
	return rep, nil
}
