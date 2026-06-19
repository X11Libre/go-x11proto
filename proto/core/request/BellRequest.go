package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// BellRequest rings the keyboard bell. Percent is -100..100 (param0, INT8).
type BellRequest struct {
	Percent int8
}

func (r *BellRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.Bell)
	writer.SetParam0(base.CARD8(r.Percent))
	return nil
}
