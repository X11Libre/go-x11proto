package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// Grab mode (pointer-mode / keyboard-mode).
const (
	GrabModeSync  = 0
	GrabModeAsync = 1
)

// Grab reply status.
const (
	GrabStatusSuccess        = 0
	GrabStatusAlreadyGrabbed = 1
	GrabStatusInvalidTime    = 2
	GrabStatusNotViewable    = 3
	GrabStatusFrozen         = 4
)

type GrabPointerRequest struct {
	OwnerEvents  bool
	GrabWindow   base.WINDOW
	EventMask    base.CARD16
	PointerMode  base.CARD8
	KeyboardMode base.CARD8
	ConfineTo    base.WINDOW
	Cursor       base.CURSOR
	Time         base.CARD32
}

func (r *GrabPointerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GrabPointer)
	writer.SetParam0bool(r.OwnerEvents)
	writer.WriteXID(r.GrabWindow)
	writer.WriteCARD16(r.EventMask)
	writer.WriteCARD8(r.PointerMode)
	writer.WriteCARD8(r.KeyboardMode)
	writer.WriteXID(r.ConfineTo)
	writer.WriteXID(r.Cursor)
	writer.WriteCARD32(r.Time)
	return nil
}

type GrabPointerReply struct {
	Status base.CARD8
}

func (reply *GrabPointerReply) Parse(reader base.ReplyReader) error {
	reply.Status = reader.Data0
	return reader.LastError
}
