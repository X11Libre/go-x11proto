package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// ChangeSaveSet mode (the param0 byte).
const (
	SaveSetInsert = 0
	SaveSetDelete = 1
)

type ChangeSaveSetRequest struct {
	Mode   base.CARD8
	Window base.WINDOW
}

func (r *ChangeSaveSetRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangeSaveSet)
	writer.SetParam0(r.Mode)
	writer.WriteXID(r.Window)
	return nil
}
