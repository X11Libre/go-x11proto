package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type CopyGCRequest struct {
	SrcGC     base.GC
	DstGC     base.GC
	ValueMask base.CARD32
}

func (r *CopyGCRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CopyGC)
	writer.WriteXID(r.SrcGC)
	writer.WriteXID(r.DstGC)
	writer.WriteCARD32(r.ValueMask)
	return nil
}
