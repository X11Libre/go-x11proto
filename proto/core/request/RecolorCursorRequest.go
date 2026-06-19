package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type RecolorCursorRequest struct {
	Cursor    base.CURSOR
	ForeRed   base.CARD16
	ForeGreen base.CARD16
	ForeBlue  base.CARD16
	BackRed   base.CARD16
	BackGreen base.CARD16
	BackBlue  base.CARD16
}

func (r *RecolorCursorRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.RecolorCursor)
	writer.WriteXID(r.Cursor)
	writer.WriteCARD16(r.ForeRed)
	writer.WriteCARD16(r.ForeGreen)
	writer.WriteCARD16(r.ForeBlue)
	writer.WriteCARD16(r.BackRed)
	writer.WriteCARD16(r.BackGreen)
	writer.WriteCARD16(r.BackBlue)
	return nil
}
