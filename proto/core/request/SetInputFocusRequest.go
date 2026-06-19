package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// SetInputFocus revert-to (the param0 byte).
const (
	RevertToNone        = 0
	RevertToPointerRoot = 1
	RevertToParent      = 2
)

type SetInputFocusRequest struct {
	RevertTo base.CARD8
	Focus    base.WINDOW
	Time     base.CARD32
}

func (r *SetInputFocusRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetInputFocus)
	writer.SetParam0(r.RevertTo)
	writer.WriteXID(r.Focus)
	writer.WriteCARD32(r.Time)
	return nil
}
