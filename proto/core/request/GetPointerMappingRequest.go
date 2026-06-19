package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetPointerMappingRequest struct{}

func (r *GetPointerMappingRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetPointerMapping)
	return nil
}

type GetPointerMappingReply struct {
	Map []byte
}

func (reply *GetPointerMappingReply) Parse(reader base.ReplyReader) error {
	n := uint(reader.Data0)
	reader.ReadBytes(24) // unused
	reply.Map = reader.ReadBytes(n)
	return reader.LastError
}
