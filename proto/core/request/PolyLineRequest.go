package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type PolyLineRequest struct {
	CoordMode base.CARD8
	Drawable  base.DRAWABLE
	Gc        base.GC
	Points    []base.Point
}

func (r *PolyLineRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.PolyLine)
	writer.SetParam0(r.CoordMode)
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	for _, p := range r.Points {
		writer.WriteINT16(p.X)
		writer.WriteINT16(p.Y)
	}
	return nil
}
