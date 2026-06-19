package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ChangePointerControlRequest struct {
	AccelerationNumerator   base.INT16
	AccelerationDenominator base.INT16
	Threshold               base.INT16
	DoAcceleration          bool
	DoThreshold             bool
}

func (r *ChangePointerControlRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangePointerControl)
	writer.WriteINT16(r.AccelerationNumerator)
	writer.WriteINT16(r.AccelerationDenominator)
	writer.WriteINT16(r.Threshold)
	writer.WriteBool(r.DoAcceleration)
	writer.WriteBool(r.DoThreshold)
	return nil
}
