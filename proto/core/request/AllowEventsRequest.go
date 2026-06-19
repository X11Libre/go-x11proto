package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// AllowEvents mode (the param0 byte).
const (
	AllowAsyncPointer   = 0
	AllowSyncPointer    = 1
	AllowReplayPointer  = 2
	AllowAsyncKeyboard  = 3
	AllowSyncKeyboard   = 4
	AllowReplayKeyboard = 5
	AllowAsyncBoth      = 6
	AllowSyncBoth       = 7
)

type AllowEventsRequest struct {
	Mode base.CARD8
	Time base.CARD32
}

func (r *AllowEventsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.AllowEvents)
	writer.SetParam0(r.Mode)
	writer.WriteCARD32(r.Time)
	return nil
}
