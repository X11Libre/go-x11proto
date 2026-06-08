package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type CreatePixmapRequest struct {
	Depth    base.CARD8
	Pid      base.PIXMAP
	Drawable base.DRAWABLE
	Width    base.CARD16
	Height   base.CARD16
}

func (r *CreatePixmapRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CreatePixmap)
	writer.SetParam0(r.Depth)
	writer.WriteXID(base.DRAWABLE(r.Pid))
	writer.WriteXID(r.Drawable)
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	return nil
}
