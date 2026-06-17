package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type DestroyWindowRequest struct {
	Window base.WINDOW
}

func (r *DestroyWindowRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.DestroyWindow)
	writer.WriteXID(base.XID(r.Window))
	return nil
}
