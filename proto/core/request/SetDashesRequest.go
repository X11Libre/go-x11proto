package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type SetDashesRequest struct {
	Gc         base.GC
	DashOffset base.CARD16
	Dashes     []base.CARD8
}

func (r *SetDashesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetDashes)
	writer.WriteXID(r.Gc)
	writer.WriteCARD16(r.DashOffset)
	writer.WriteCARD16(base.CARD16(len(r.Dashes)))
	writer.WriteCARD8s(r.Dashes)
	writer.Pad()
	return nil
}
