package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetKeyboardMappingRequest struct {
	FirstKeycode base.CARD8
	Count        base.CARD8
}

func (r *GetKeyboardMappingRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetKeyboardMap)
	writer.WriteCARD8(r.FirstKeycode)
	writer.WriteCARD8(r.Count)
	writer.WriteCARD16(0) // unused
	return nil
}

type GetKeyboardMappingReply struct {
	KeysymsPerKeycode base.CARD8
	Keysyms           []base.CARD32
}

func (reply *GetKeyboardMappingReply) Parse(reader base.ReplyReader) error {
	reply.KeysymsPerKeycode = reader.Data0
	reader.ReadBytes(24) // unused
	n := int(reader.Length)
	reply.Keysyms = make([]base.CARD32, 0, n)
	for i := 0; i < n; i++ {
		reply.Keysyms = append(reply.Keysyms, reader.CARD32())
	}
	return reader.LastError
}
