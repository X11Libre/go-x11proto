package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type UngrabButtonRequest struct {
	Button     base.CARD8
	GrabWindow base.WINDOW
	Modifiers  base.CARD16
}

func (r *UngrabButtonRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.UngrabButton)
	writer.SetParam0(r.Button)
	writer.WriteXID(r.GrabWindow)
	writer.WriteCARD16(r.Modifiers)
	writer.WriteCARD16(0) // unused
	return nil
}
