package request

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// ImageText16 draws 2-byte (CHAR2B) characters. Each char is written
// most-significant byte first, regardless of connection byte order.
type ImageText16Request struct {
	Drawable base.DRAWABLE
	Gc       base.GC
	X        base.INT16
	Y        base.INT16
	Text     []base.CARD16
}

func (r *ImageText16Request) WriteInto(writer *base.RequestWriter) error {
	if len(r.Text) > 255 {
		return fmt.Errorf("ImageText16Request: text too long")
	}
	writer.SetOpcode(opcode.ImageText16)
	writer.SetParam0(base.CARD8(len(r.Text)))
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)
	for _, ch := range r.Text {
		writer.WriteCARD8(base.CARD8(ch >> 8))
		writer.WriteCARD8(base.CARD8(ch & 0xff))
	}
	writer.Pad()
	return nil
}
