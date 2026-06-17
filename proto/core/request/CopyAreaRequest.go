package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type CopyAreaRequest struct {
	SrcDrawable base.DRAWABLE
	DstDrawable base.DRAWABLE
	GC          base.GC
	SrcX        base.INT16
	SrcY        base.INT16
	DstX        base.INT16
	DstY        base.INT16
	Width       base.CARD16
	Height      base.CARD16
}

func (r *CopyAreaRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CopyArea)
	writer.WriteXID(base.XID(r.SrcDrawable))
	writer.WriteXID(base.XID(r.DstDrawable))
	writer.WriteXID(base.XID(r.GC))
	writer.WriteINT16(r.SrcX)
	writer.WriteINT16(r.SrcY)
	writer.WriteINT16(r.DstX)
	writer.WriteINT16(r.DstY)
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	return nil
}
