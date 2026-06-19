package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type RotatePropertiesRequest struct {
	Window     base.WINDOW
	Delta      base.INT16
	Properties []base.ATOM
}

func (r *RotatePropertiesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.RotateProperties)
	writer.WriteXID(r.Window)
	writer.WriteCARD16(base.CARD16(len(r.Properties)))
	writer.WriteINT16(r.Delta)
	for _, a := range r.Properties {
		writer.WriteATOM(a)
	}
	return nil
}
