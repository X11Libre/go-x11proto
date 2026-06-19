package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// FillPoly shape hint.
const (
	PolyShapeComplex   = 0
	PolyShapeNonconvex = 1
	PolyShapeConvex    = 2
)

type FillPolyRequest struct {
	Drawable  base.DRAWABLE
	Gc        base.GC
	Shape     base.CARD8
	CoordMode base.CARD8
	Points    []base.Point
}

func (r *FillPolyRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.FillPoly)
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	writer.WriteCARD8(r.Shape)
	writer.WriteCARD8(r.CoordMode)
	writer.WriteCARD16(0) // pad
	for _, p := range r.Points {
		writer.WriteINT16(p.X)
		writer.WriteINT16(p.Y)
	}
	return nil
}
