package rpc

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
)

func CopyGC(c *core.X11Conn, src, dst base.GC, valueMask base.CARD32) error {
	_, err := c.Send(&request.CopyGCRequest{SrcGC: src, DstGC: dst, ValueMask: valueMask})
	return err
}
