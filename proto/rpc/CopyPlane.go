package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CopyPlane(c *core.X11Conn, src, dst base.DRAWABLE, gc base.GC, srcX, srcY, dstX, dstY base.INT16, width, height base.CARD16, bitPlane base.CARD32) error {
	_, err := c.Send(&request.CopyPlaneRequest{
		SrcDrawable: src, DstDrawable: dst, Gc: gc,
		SrcX: srcX, SrcY: srcY, DstX: dstX, DstY: dstY,
		Width: width, Height: height, BitPlane: bitPlane,
	})
	return err
}
