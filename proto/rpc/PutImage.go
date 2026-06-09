package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func PutImage(c *core.X11Conn, drawable base.DRAWABLE, gc base.GC, format base.CARD8, depth base.CARD8, width base.CARD16, height base.CARD16, dstX, dstY base.INT16, data []byte) error {
	req := request.PutImageRequest{
		Format:   format,
		Drawable: drawable,
		Gc:       gc,
		Width:    width,
		Height:   height,
		DstX:     dstX,
		DstY:     dstY,
		LeftPad:  0,
		Depth:    depth,
		Data:     data,
	}
	_, err := c.Send(&req)
	return err
}
