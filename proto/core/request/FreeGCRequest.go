package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type FreeGCRequest struct {
	Gc base.GC
}

func (r *FreeGCRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.FreeGC)
	writer.WriteXID(r.Gc)
	return nil
}
