package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type CopyPlaneRequest struct {
	SrcDrawable base.DRAWABLE
	DstDrawable base.DRAWABLE
	Gc          base.GC
	SrcX        base.INT16
	SrcY        base.INT16
	DstX        base.INT16
	DstY        base.INT16
	Width       base.CARD16
	Height      base.CARD16
	BitPlane    base.CARD32
}

func (r *CopyPlaneRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CopyPlane)
	writer.WriteXID(r.SrcDrawable)
	writer.WriteXID(r.DstDrawable)
	writer.WriteXID(r.Gc)
	writer.WriteINT16(r.SrcX)
	writer.WriteINT16(r.SrcY)
	writer.WriteINT16(r.DstX)
	writer.WriteINT16(r.DstY)
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	writer.WriteCARD32(r.BitPlane)
	return nil
}
