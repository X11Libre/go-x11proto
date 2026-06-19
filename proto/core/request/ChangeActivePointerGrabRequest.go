package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ChangeActivePointerGrabRequest struct {
	Cursor    base.CURSOR
	Time      base.CARD32
	EventMask base.CARD16
}

func (r *ChangeActivePointerGrabRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangeActivePointer)
	writer.WriteXID(r.Cursor)
	writer.WriteCARD32(r.Time)
	writer.WriteCARD16(r.EventMask)
	writer.WriteCARD16(0) // unused
	return nil
}
