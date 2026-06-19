package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type UngrabKeyRequest struct {
	Key        base.CARD8 // keycode
	GrabWindow base.WINDOW
	Modifiers  base.CARD16
}

func (r *UngrabKeyRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.UngrabKey)
	writer.SetParam0(r.Key)
	writer.WriteXID(r.GrabWindow)
	writer.WriteCARD16(r.Modifiers)
	writer.WriteCARD16(0) // unused
	return nil
}
