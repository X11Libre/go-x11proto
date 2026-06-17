package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type SendEventRequest struct {
	Propagate   bool
	Destination base.WINDOW
	EventMask   base.CARD32
	Event       [32]byte // the 32-byte event to deliver
}

func (r *SendEventRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SendEvent)
	writer.SetParam0bool(r.Propagate)
	writer.WriteXID(base.XID(r.Destination))
	writer.WriteCARD32(r.EventMask)
	writer.WriteBytes(r.Event[:])
	return nil
}
