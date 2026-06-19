package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type CreateGlyphCursorRequest struct {
	Cid        base.CURSOR
	SourceFont base.FONT
	MaskFont   base.FONT
	SourceChar base.CARD16
	MaskChar   base.CARD16
	ForeRed    base.CARD16
	ForeGreen  base.CARD16
	ForeBlue   base.CARD16
	BackRed    base.CARD16
	BackGreen  base.CARD16
	BackBlue   base.CARD16
}

func (r *CreateGlyphCursorRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CreateGlyphCursor)
	writer.WriteXID(r.Cid)
	writer.WriteXID(r.SourceFont)
	writer.WriteXID(r.MaskFont)
	writer.WriteCARD16(r.SourceChar)
	writer.WriteCARD16(r.MaskChar)
	writer.WriteCARD16(r.ForeRed)
	writer.WriteCARD16(r.ForeGreen)
	writer.WriteCARD16(r.ForeBlue)
	writer.WriteCARD16(r.BackRed)
	writer.WriteCARD16(r.BackGreen)
	writer.WriteCARD16(r.BackBlue)
	return nil
}
