package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetGeometryRequest struct {
	Drawable base.DRAWABLE
}

func (r *GetGeometryRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetGeometry)
	writer.WriteXID(r.Drawable)
	return nil
}

type GetGeometryReply struct {
	Depth       base.CARD8
	Root        base.WINDOW
	X           base.INT16
	Y           base.INT16
	Width       base.CARD16
	Height      base.CARD16
	BorderWidth base.CARD16
}

func (reply *GetGeometryReply) Parse(reader base.ReplyReader) error {
	reply.Depth = reader.Data0 // depth is carried in the reply header byte
	reply.Root = reader.XID()
	reply.X = reader.INT16()
	reply.Y = reader.INT16()
	reply.Width = reader.CARD16()
	reply.Height = reader.CARD16()
	reply.BorderWidth = reader.CARD16()
	return reader.LastError
}
