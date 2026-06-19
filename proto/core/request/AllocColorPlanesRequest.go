package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type AllocColorPlanesRequest struct {
	Contiguous bool
	Colormap   base.COLORMAP
	Colors     base.CARD16
	Reds       base.CARD16
	Greens     base.CARD16
	Blues      base.CARD16
}

func (r *AllocColorPlanesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.AllocColorPlanes)
	writer.SetParam0bool(r.Contiguous)
	writer.WriteXID(r.Colormap)
	writer.WriteCARD16(r.Colors)
	writer.WriteCARD16(r.Reds)
	writer.WriteCARD16(r.Greens)
	writer.WriteCARD16(r.Blues)
	return nil
}

type AllocColorPlanesReply struct {
	RedMask   base.CARD32
	GreenMask base.CARD32
	BlueMask  base.CARD32
	Pixels    []base.CARD32
}

func (reply *AllocColorPlanesReply) Parse(reader base.ReplyReader) error {
	n := int(reader.CARD16())
	reader.ReadBytes(2) // unused
	reply.RedMask = reader.CARD32()
	reply.GreenMask = reader.CARD32()
	reply.BlueMask = reader.CARD32()
	reader.ReadBytes(8) // unused
	reply.Pixels = make([]base.CARD32, 0, n)
	for i := 0; i < n; i++ {
		reply.Pixels = append(reply.Pixels, reader.CARD32())
	}
	return reader.LastError
}
