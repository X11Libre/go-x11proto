package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type PolyRectangleRequest struct {
	Drawable   base.DRAWABLE
	Gc         base.GC
	Rectangles []base.Rectangle
}

func (r *PolyRectangleRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.PolyRectangle)
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	for _, rect := range r.Rectangles {
		writer.WriteINT16(rect.X)
		writer.WriteINT16(rect.Y)
		writer.WriteCARD16(rect.Width)
		writer.WriteCARD16(rect.Height)
	}
	return nil
}
