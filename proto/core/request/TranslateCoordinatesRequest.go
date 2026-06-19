package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type TranslateCoordinatesRequest struct {
	SrcWindow base.WINDOW
	DstWindow base.WINDOW
	SrcX      base.INT16
	SrcY      base.INT16
}

func (r *TranslateCoordinatesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.TranslateCoords)
	writer.WriteXID(r.SrcWindow)
	writer.WriteXID(r.DstWindow)
	writer.WriteINT16(r.SrcX)
	writer.WriteINT16(r.SrcY)
	return nil
}

type TranslateCoordinatesReply struct {
	SameScreen bool
	Child      base.WINDOW
	DstX       base.INT16
	DstY       base.INT16
}

func (reply *TranslateCoordinatesReply) Parse(reader base.ReplyReader) error {
	reply.SameScreen = reader.Data0 != 0
	reply.Child = reader.XID()
	reply.DstX = reader.INT16()
	reply.DstY = reader.INT16()
	return reader.LastError
}
