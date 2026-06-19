package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type LookupColorRequest struct {
	Colormap base.COLORMAP
	Name     string
}

func (r *LookupColorRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.LookupColor)
	writer.WriteXID(r.Colormap)
	writer.WriteCARD16(base.CARD16(len(r.Name)))
	writer.WriteCARD16(0) // unused
	writer.WriteBytes([]byte(r.Name))
	writer.Pad()
	return nil
}

type LookupColorReply struct {
	ExactRed    base.CARD16
	ExactGreen  base.CARD16
	ExactBlue   base.CARD16
	VisualRed   base.CARD16
	VisualGreen base.CARD16
	VisualBlue  base.CARD16
}

func (reply *LookupColorReply) Parse(reader base.ReplyReader) error {
	reply.ExactRed = reader.CARD16()
	reply.ExactGreen = reader.CARD16()
	reply.ExactBlue = reader.CARD16()
	reply.VisualRed = reader.CARD16()
	reply.VisualGreen = reader.CARD16()
	reply.VisualBlue = reader.CARD16()
	return reader.LastError
}
