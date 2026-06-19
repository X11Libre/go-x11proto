package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ConvertSelectionRequest struct {
	Requestor base.WINDOW
	Selection base.ATOM
	Target    base.ATOM
	Property  base.ATOM
	Time      base.CARD32
}

func (r *ConvertSelectionRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ConvertSelection)
	writer.WriteXID(r.Requestor)
	writer.WriteATOM(r.Selection)
	writer.WriteATOM(r.Target)
	writer.WriteATOM(r.Property)
	writer.WriteCARD32(r.Time)
	return nil
}
