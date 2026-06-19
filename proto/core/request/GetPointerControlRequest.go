package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetPointerControlRequest struct{}

func (r *GetPointerControlRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetPointerControl)
	return nil
}

type GetPointerControlReply struct {
	AccelerationNumerator   base.CARD16
	AccelerationDenominator base.CARD16
	Threshold               base.CARD16
}

func (reply *GetPointerControlReply) Parse(reader base.ReplyReader) error {
	reply.AccelerationNumerator = reader.CARD16()
	reply.AccelerationDenominator = reader.CARD16()
	reply.Threshold = reader.CARD16()
	return reader.LastError
}
