package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GrabKeyboardRequest struct {
	OwnerEvents  bool
	GrabWindow   base.WINDOW
	Time         base.CARD32
	PointerMode  base.CARD8
	KeyboardMode base.CARD8
}

func (r *GrabKeyboardRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GrabKeyboard)
	writer.SetParam0bool(r.OwnerEvents)
	writer.WriteXID(r.GrabWindow)
	writer.WriteCARD32(r.Time)
	writer.WriteCARD8(r.PointerMode)
	writer.WriteCARD8(r.KeyboardMode)
	writer.WriteCARD16(0) // unused
	return nil
}

type GrabKeyboardReply struct {
	Status base.CARD8
}

func (reply *GrabKeyboardReply) Parse(reader base.ReplyReader) error {
	reply.Status = reader.Data0
	return reader.LastError
}
