package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type PolyFillRectRequest struct {
	Drawable base.DRAWABLE
	Gc       base.GC
	Rects    []base.Rectangle
}

func (r *PolyFillRectRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.PolyFillRectangle)
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	for _, rect := range r.Rects {
		rect.WriteInto(writer)
	}
	return nil
}
