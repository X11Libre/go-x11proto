package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CopyArea(c *core.X11Conn, src, dst base.DRAWABLE, gc base.GC, srcX, srcY, dstX, dstY base.INT16, width, height base.CARD16) error {
	_, err := c.Send(&request.CopyAreaRequest{
		SrcDrawable: src, DstDrawable: dst, GC: gc,
		SrcX: srcX, SrcY: srcY, DstX: dstX, DstY: dstY,
		Width: width, Height: height,
	})
	return err
}
