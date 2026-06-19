package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// SetClipRectangles ordering (the param0 byte).
const (
	ClipOrderingUnsorted = 0
	ClipOrderingYSorted  = 1
	ClipOrderingYXSorted = 2
	ClipOrderingYXBanded = 3
)

type SetClipRectanglesRequest struct {
	Ordering    base.CARD8
	Gc          base.GC
	ClipXOrigin base.INT16
	ClipYOrigin base.INT16
	Rectangles  []base.Rectangle
}

func (r *SetClipRectanglesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetClipRect)
	writer.SetParam0(r.Ordering)
	writer.WriteXID(r.Gc)
	writer.WriteINT16(r.ClipXOrigin)
	writer.WriteINT16(r.ClipYOrigin)
	for _, rect := range r.Rectangles {
		writer.WriteINT16(rect.X)
		writer.WriteINT16(rect.Y)
		writer.WriteCARD16(rect.Width)
		writer.WriteCARD16(rect.Height)
	}
	return nil
}
