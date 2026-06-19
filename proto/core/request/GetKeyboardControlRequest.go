package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetKeyboardControlRequest struct{}

func (r *GetKeyboardControlRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetKeyboardControl)
	return nil
}

type GetKeyboardControlReply struct {
	GlobalAutoRepeat bool
	LedMask          base.CARD32
	KeyClickPercent  base.CARD8
	BellPercent      base.CARD8
	BellPitch        base.CARD16
	BellDuration     base.CARD16
	AutoRepeats      []byte // 32-byte per-key auto-repeat bitmap
}

func (reply *GetKeyboardControlReply) Parse(reader base.ReplyReader) error {
	reply.GlobalAutoRepeat = reader.Data0 != 0
	reply.LedMask = reader.CARD32()
	reply.KeyClickPercent = reader.CARD8()
	reply.BellPercent = reader.CARD8()
	reply.BellPitch = reader.CARD16()
	reply.BellDuration = reader.CARD16()
	reader.ReadBytes(2) // unused
	reply.AutoRepeats = reader.ReadBytes(32)
	return reader.LastError
}
