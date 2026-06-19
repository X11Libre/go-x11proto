package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// SetModifierMapping sets the keycodes for the 8 modifiers; Keycodes is
// laid out as 8 rows of KeycodesPerModifier each.
type SetModifierMappingRequest struct {
	KeycodesPerModifier base.CARD8
	Keycodes            []base.CARD8
}

func (r *SetModifierMappingRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetModifierMapping)
	writer.SetParam0(r.KeycodesPerModifier)
	writer.WriteCARD8s(r.Keycodes)
	writer.Pad()
	return nil
}

type SetModifierMappingReply struct {
	Status base.CARD8
}

func (reply *SetModifierMappingReply) Parse(reader base.ReplyReader) error {
	reply.Status = reader.Data0
	return reader.LastError
}
