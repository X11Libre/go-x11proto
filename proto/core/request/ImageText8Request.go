package request

import (
	"fmt"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ImageText8Request struct {
	Drawable base.DRAWABLE
	Gc       base.GC
	X        base.INT16
	Y        base.INT16
	Text     string
}

func (r *ImageText8Request) WriteInto(writer *base.RequestWriter) error {
	if len(r.Text) > 255 {
		return fmt.Errorf("ImageText8Request: text too long")
	}
	writer.SetOpcode(opcode.ImageText8)
	writer.SetParam0(base.CARD8(len(r.Text)))
	writer.WriteXID(r.Drawable)
	writer.WriteXID(r.Gc)
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)
	writer.WriteBytes([]byte(r.Text))
	writer.Pad()
	return nil
}
