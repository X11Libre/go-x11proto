package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type AllocColorRequest struct {
	Colormap base.COLORMAP
	Red      base.CARD16
	Green    base.CARD16
	Blue     base.CARD16
}

func (r *AllocColorRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.AllocColor)
	writer.WriteXID(r.Colormap)
	writer.WriteCARD16(r.Red)
	writer.WriteCARD16(r.Green)
	writer.WriteCARD16(r.Blue)
	writer.WriteCARD16(0) // unused
	return nil
}

type AllocColorReply struct {
	Red   base.CARD16
	Green base.CARD16
	Blue  base.CARD16
	Pixel base.CARD32
}

func (reply *AllocColorReply) Parse(reader base.ReplyReader) error {
	reply.Red = reader.CARD16()
	reply.Green = reader.CARD16()
	reply.Blue = reader.CARD16()
	reader.ReadBytes(2) // unused
	reply.Pixel = reader.CARD32()
	return reader.LastError
}
