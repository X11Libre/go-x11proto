package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type MapSubwindowsRequest struct {
	Window base.WINDOW
}

func (r *MapSubwindowsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.MapSubwindows)
	writer.WriteXID(r.Window)
	return nil
}
