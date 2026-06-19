package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type DestroySubwindowsRequest struct {
	Window base.WINDOW
}

func (r *DestroySubwindowsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.DestroySubwindows)
	writer.WriteXID(r.Window)
	return nil
}
