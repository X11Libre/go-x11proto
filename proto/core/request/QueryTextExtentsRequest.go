package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// QueryTextExtents measures a CHAR2B string. Font is a FONTABLE (FONT or GC).
type QueryTextExtentsRequest struct {
	Font base.FONT
	Text []base.CARD16
}

func (r *QueryTextExtentsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryTextExtents)
	if len(r.Text)%2 != 0 {
		writer.SetParam0(1) // odd-length
	}
	writer.WriteXID(r.Font)
	for _, ch := range r.Text {
		writer.WriteCARD8(base.CARD8(ch >> 8))
		writer.WriteCARD8(base.CARD8(ch & 0xff))
	}
	writer.Pad()
	return nil
}

type QueryTextExtentsReply struct {
	DrawDirection  base.CARD8
	FontAscent     base.INT16
	FontDescent    base.INT16
	OverallAscent  base.INT16
	OverallDescent base.INT16
	OverallWidth   int32
	OverallLeft    int32
	OverallRight   int32
}

func (reply *QueryTextExtentsReply) Parse(reader base.ReplyReader) error {
	reply.DrawDirection = reader.Data0
	reply.FontAscent = reader.INT16()
	reply.FontDescent = reader.INT16()
	reply.OverallAscent = reader.INT16()
	reply.OverallDescent = reader.INT16()
	reply.OverallWidth = int32(reader.CARD32())
	reply.OverallLeft = int32(reader.CARD32())
	reply.OverallRight = int32(reader.CARD32())
	return reader.LastError
}
