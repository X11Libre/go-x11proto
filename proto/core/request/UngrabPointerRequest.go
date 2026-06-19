package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type UngrabPointerRequest struct {
	Time base.CARD32
}

func (r *UngrabPointerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.UngrabPointer)
	writer.WriteCARD32(r.Time)
	return nil
}
