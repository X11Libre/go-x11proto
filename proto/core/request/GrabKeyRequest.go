package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GrabKeyRequest struct {
	OwnerEvents  bool
	GrabWindow   base.WINDOW
	Modifiers    base.CARD16
	Key          base.CARD8 // keycode
	PointerMode  base.CARD8
	KeyboardMode base.CARD8
}

func (r *GrabKeyRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GrabKey)
	writer.SetParam0bool(r.OwnerEvents)
	writer.WriteXID(r.GrabWindow)
	writer.WriteCARD16(r.Modifiers)
	writer.WriteCARD8(r.Key)
	writer.WriteCARD8(r.PointerMode)
	writer.WriteCARD8(r.KeyboardMode)
	writer.WriteCARD8(0)  // unused
	writer.WriteCARD16(0) // unused
	return nil
}
