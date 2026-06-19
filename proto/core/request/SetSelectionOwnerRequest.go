package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type SetSelectionOwnerRequest struct {
	Owner     base.WINDOW
	Selection base.ATOM
	Time      base.CARD32
}

func (r *SetSelectionOwnerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetSelectionOwner)
	writer.WriteXID(r.Owner)
	writer.WriteATOM(r.Selection)
	writer.WriteCARD32(r.Time)
	return nil
}
