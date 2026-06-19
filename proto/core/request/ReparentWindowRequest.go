package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ReparentWindowRequest struct {
	Window base.WINDOW
	Parent base.WINDOW
	X      base.INT16
	Y      base.INT16
}

func (r *ReparentWindowRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ReparentWindow)
	writer.WriteXID(r.Window)
	writer.WriteXID(r.Parent)
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)
	return nil
}
