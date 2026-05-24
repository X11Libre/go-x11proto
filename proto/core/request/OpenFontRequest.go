package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type OpenFontRequest struct {
	FontID base.FONT
	Name   string
}

func (r *OpenFontRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.OpenFont)
	writer.WriteXID(base.XID(r.FontID))
	writer.WriteCARD16(base.CARD16(len(r.Name)))
	writer.WriteCARD16(0) // padding
	writer.WriteBytes([]byte(r.Name))
	writer.Pad()
	return nil
}
