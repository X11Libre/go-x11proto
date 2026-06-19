package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type UnmapSubwindowsRequest struct {
	Window base.WINDOW
}

func (r *UnmapSubwindowsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.UnmapSubwindows)
	writer.WriteXID(r.Window)
	return nil
}
