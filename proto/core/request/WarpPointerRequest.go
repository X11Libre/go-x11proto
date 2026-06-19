package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type WarpPointerRequest struct {
	SrcWindow base.WINDOW
	DstWindow base.WINDOW
	SrcX      base.INT16
	SrcY      base.INT16
	SrcWidth  base.CARD16
	SrcHeight base.CARD16
	DstX      base.INT16
	DstY      base.INT16
}

func (r *WarpPointerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.WarpPointer)
	writer.WriteXID(r.SrcWindow)
	writer.WriteXID(r.DstWindow)
	writer.WriteINT16(r.SrcX)
	writer.WriteINT16(r.SrcY)
	writer.WriteCARD16(r.SrcWidth)
	writer.WriteCARD16(r.SrcHeight)
	writer.WriteINT16(r.DstX)
	writer.WriteINT16(r.DstY)
	return nil
}
