package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// Mapping change status (returned by Set{Pointer,Modifier}Mapping).
const (
	MappingStatusSuccess = 0
	MappingStatusBusy    = 1
	MappingStatusFailed  = 2
)

type SetPointerMappingRequest struct {
	Map []base.CARD8
}

func (r *SetPointerMappingRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetPointerMapping)
	writer.SetParam0(base.CARD8(len(r.Map)))
	writer.WriteCARD8s(r.Map)
	writer.Pad()
	return nil
}

type SetPointerMappingReply struct {
	Status base.CARD8
}

func (reply *SetPointerMappingReply) Parse(reader base.ReplyReader) error {
	reply.Status = reader.Data0
	return reader.LastError
}
