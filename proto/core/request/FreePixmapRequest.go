package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type FreePixmapRequest struct {
	Pixmap base.PIXMAP
}

func (r *FreePixmapRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.FreePixmap)
	writer.WriteXID(base.XID(r.Pixmap))
	return nil
}
