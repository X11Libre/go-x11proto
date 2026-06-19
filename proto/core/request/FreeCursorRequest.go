package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type FreeCursorRequest struct {
	Cursor base.CURSOR
}

func (r *FreeCursorRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.FreeCursor)
	writer.WriteXID(r.Cursor)
	return nil
}
