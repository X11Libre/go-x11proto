package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type QueryExtensionRequest struct {
	Name string
}

func (r *QueryExtensionRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryExtension)

	writer.WriteCARD16(base.CARD16(len(r.Name)))
	writer.WriteCARD16(0) // pad0
	writer.WriteBytes([]byte(r.Name))
	return nil
}

type QueryExtensionReply struct {
	Present     bool
	MajorOpcode base.CARD8
	FirstEvent  base.CARD8
	FirstError  base.CARD8
}

func (reply *QueryExtensionReply) Parse(reader base.ReplyReader) error {
	// present/major-opcode/first-event/first-error are the first four payload
	// bytes (reply offsets 8..11); byte 1 (Data0) is unused in this reply.
	reply.Present = reader.CARD8() != 0
	reply.MajorOpcode = reader.CARD8()
	reply.FirstEvent = reader.CARD8()
	reply.FirstError = reader.CARD8()
	return reader.LastError
}
