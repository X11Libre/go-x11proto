package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// Circulate direction (the param0 byte of CirculateWindow).
const (
	CirculateRaiseLowest  = 0
	CirculateLowerHighest = 1
)

type CirculateWindowRequest struct {
	Direction base.CARD8
	Window    base.WINDOW
}

func (r *CirculateWindowRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CirculateWindow)
	writer.SetParam0(r.Direction)
	writer.WriteXID(r.Window)
	return nil
}
