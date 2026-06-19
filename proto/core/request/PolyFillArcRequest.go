package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type PolyFillArcRequest struct {
	Drawable base.DRAWABLE
	Gc       base.GC
	Arcs     []base.Arc
}

func (r *PolyFillArcRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.PolyFillArc)
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	for _, a := range r.Arcs {
		writer.WriteINT16(a.X)
		writer.WriteINT16(a.Y)
		writer.WriteCARD16(a.Width)
		writer.WriteCARD16(a.Height)
		writer.WriteINT16(a.Angle1)
		writer.WriteINT16(a.Angle2)
	}
	return nil
}
