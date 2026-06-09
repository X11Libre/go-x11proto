package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type MapWindowRequest struct {
	Window base.WINDOW
}

func (r *MapWindowRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.MapWindow)
	writer.WriteXID(r.Window)
	return nil
}
