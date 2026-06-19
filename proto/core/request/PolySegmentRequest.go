package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type PolySegmentRequest struct {
	Drawable base.DRAWABLE
	Gc       base.GC
	Segments []base.Segment
}

func (r *PolySegmentRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.PolySegment)
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	for _, s := range r.Segments {
		writer.WriteINT16(s.X1)
		writer.WriteINT16(s.Y1)
		writer.WriteINT16(s.X2)
		writer.WriteINT16(s.Y2)
	}
	return nil
}
