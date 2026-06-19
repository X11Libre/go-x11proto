package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type UngrabKeyboardRequest struct {
	Time base.CARD32
}

func (r *UngrabKeyboardRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.UngrabKeyboard)
	writer.WriteCARD32(r.Time)
	return nil
}
