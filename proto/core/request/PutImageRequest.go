package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

const (
	PutImageFormat_XYBitmap = 0
	PutImageFormat_XYPixmap = 1
	PutImageFormat_ZPixmap  = 2
)

type PutImageRequest struct {
	Format   base.CARD8
	Drawable base.DRAWABLE
	Gc       base.GC
	Width    base.CARD16
	Height   base.CARD16
	DstX     base.INT16
	DstY     base.INT16
	LeftPad  base.CARD8
	Depth    base.CARD8
	Data     []byte
}

func (r *PutImageRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.PutImage)
	writer.SetParam0(r.Format)
	writer.WriteXID(base.XID(r.Drawable))
	writer.WriteXID(base.XID(r.Gc))
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	writer.WriteINT16(r.DstX)
	writer.WriteINT16(r.DstY)
	writer.WriteCARD8(r.LeftPad)
	writer.WriteCARD8(r.Depth)
	writer.WriteCARD8(0)
	writer.WriteCARD8(0)
	writer.WriteBytes(r.Data)
	writer.Pad()
	return nil
}
