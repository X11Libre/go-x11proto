package request

import (
	"fmt"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// FIXME: not supporting intermedia font change yet, just pure text
type PutText8Request struct {
	Drawable base.DRAWABLE
	Gc       base.GC
	X        base.INT16
	Y        base.INT16
	Text     string
}

const (
	MAX_STR_ELEM_SIZE = 254
)

func (r *PutText8Request) WriteInto(writer *base.RequestWriter) error {

	l := len(r.Text)
	if l > MAX_STR_ELEM_SIZE {
		return fmt.Errorf("PutText8Request: text too long\n")
	}

	writer.SetOpcode(opcode.PutText8)
	writer.SetParam0(base.CARD8(l))
	writer.WriteXID(base.XID(r.Drawable))
	writer.WriteXID(base.XID(r.Gc))
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)

	writer.WriteCARD8(base.CARD8(l))
	writer.WriteCARD8(0)
	writer.WriteBytes([]byte(r.Text))

	writer.Pad()
	return nil
}
