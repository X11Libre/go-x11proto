package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type DeletePropertyRequest struct {
	Window   base.WINDOW
	Property base.ATOM
}

func (r *DeletePropertyRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.DeleteProperty)
	writer.WriteXID(r.Window)
	writer.WriteATOM(r.Property)
	return nil
}
