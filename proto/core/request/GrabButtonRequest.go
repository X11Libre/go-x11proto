package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GrabButtonRequest struct {
	OwnerEvents  bool
	GrabWindow   base.WINDOW
	EventMask    base.CARD16
	PointerMode  base.CARD8
	KeyboardMode base.CARD8
	ConfineTo    base.WINDOW
	Cursor       base.CURSOR
	Button       base.CARD8
	Modifiers    base.CARD16
}

func (r *GrabButtonRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GrabButton)
	writer.SetParam0bool(r.OwnerEvents)
	writer.WriteXID(r.GrabWindow)
	writer.WriteCARD16(r.EventMask)
	writer.WriteCARD8(r.PointerMode)
	writer.WriteCARD8(r.KeyboardMode)
	writer.WriteXID(r.ConfineTo)
	writer.WriteXID(r.Cursor)
	writer.WriteCARD8(r.Button)
	writer.WriteCARD8(0) // unused
	writer.WriteCARD16(r.Modifiers)
	return nil
}
