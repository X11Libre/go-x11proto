package request

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// PolyText16 draws 2-byte (CHAR2B) text as a single text item.
// FIXME: intermediate font changes / multiple items are not supported yet.
type PolyText16Request struct {
	Drawable base.DRAWABLE
	Gc       base.GC
	X        base.INT16
	Y        base.INT16
	Text     []base.CARD16
}

func (r *PolyText16Request) WriteInto(writer *base.RequestWriter) error {
	if len(r.Text) > MAX_STR_ELEM_SIZE {
		return fmt.Errorf("PolyText16Request: text too long")
	}
	writer.SetOpcode(opcode.PolyText16)
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)
	// single text item: length, delta, then the CHAR2B characters
	writer.WriteCARD8(base.CARD8(len(r.Text)))
	writer.WriteCARD8(0)
	for _, ch := range r.Text {
		writer.WriteCARD8(base.CARD8(ch >> 8))
		writer.WriteCARD8(base.CARD8(ch & 0xff))
	}
	writer.Pad()
	return nil
}
