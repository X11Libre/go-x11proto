package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ClearAreaRequest struct {
	Window    base.WINDOW
	X         base.INT16
	Y         base.INT16
	Width     base.CARD16
	Height    base.CARD16
	Exposures bool
}

func (r *ClearAreaRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ClearArea)
	writer.WriteXID(r.Window)
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	if r.Exposures {
		writer.WriteCARD8(1)
	} else {
		writer.WriteCARD8(0)
	}
	writer.Pad()
	return nil
}
