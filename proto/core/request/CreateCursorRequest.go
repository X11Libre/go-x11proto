package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type CreateCursorRequest struct {
	Cid       base.CURSOR
	Source    base.PIXMAP
	Mask      base.PIXMAP
	ForeRed   base.CARD16
	ForeGreen base.CARD16
	ForeBlue  base.CARD16
	BackRed   base.CARD16
	BackGreen base.CARD16
	BackBlue  base.CARD16
	X         base.CARD16
	Y         base.CARD16
}

func (r *CreateCursorRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CreateCursor)
	writer.WriteXID(r.Cid)
	writer.WriteXID(r.Source)
	writer.WriteXID(r.Mask)
	writer.WriteCARD16(r.ForeRed)
	writer.WriteCARD16(r.ForeGreen)
	writer.WriteCARD16(r.ForeBlue)
	writer.WriteCARD16(r.BackRed)
	writer.WriteCARD16(r.BackGreen)
	writer.WriteCARD16(r.BackBlue)
	writer.WriteCARD16(r.X)
	writer.WriteCARD16(r.Y)
	return nil
}
