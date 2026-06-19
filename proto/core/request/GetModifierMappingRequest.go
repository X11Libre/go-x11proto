package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetModifierMappingRequest struct{}

func (r *GetModifierMappingRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetModifierMapping)
	return nil
}

type GetModifierMappingReply struct {
	KeycodesPerModifier base.CARD8
	Keycodes            []base.CARD8 // 8 * KeycodesPerModifier entries
}

func (reply *GetModifierMappingReply) Parse(reader base.ReplyReader) error {
	reply.KeycodesPerModifier = reader.Data0
	reader.ReadBytes(24) // unused
	n := uint(reply.KeycodesPerModifier) * 8
	b := reader.ReadBytes(n)
	reply.Keycodes = make([]base.CARD8, len(b))
	for i, v := range b {
		reply.Keycodes[i] = base.CARD8(v)
	}
	return reader.LastError
}
