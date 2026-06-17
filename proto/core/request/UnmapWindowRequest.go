package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type UnmapWindowRequest struct {
	Window base.WINDOW
}

func (r *UnmapWindowRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.UnmapWindow)
	writer.WriteXID(r.Window)
	return nil
}
