package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type AllocColorCellsRequest struct {
	Contiguous bool
	Colormap   base.COLORMAP
	Colors     base.CARD16
	Planes     base.CARD16
}

func (r *AllocColorCellsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.AllocColorCells)
	writer.SetParam0bool(r.Contiguous)
	writer.WriteXID(r.Colormap)
	writer.WriteCARD16(r.Colors)
	writer.WriteCARD16(r.Planes)
	return nil
}

type AllocColorCellsReply struct {
	Pixels []base.CARD32
	Masks  []base.CARD32
}

func (reply *AllocColorCellsReply) Parse(reader base.ReplyReader) error {
	nPixels := int(reader.CARD16())
	nMasks := int(reader.CARD16())
	reader.ReadBytes(20) // unused
	reply.Pixels = make([]base.CARD32, 0, nPixels)
	for i := 0; i < nPixels; i++ {
		reply.Pixels = append(reply.Pixels, reader.CARD32())
	}
	reply.Masks = make([]base.CARD32, 0, nMasks)
	for i := 0; i < nMasks; i++ {
		reply.Masks = append(reply.Masks, reader.CARD32())
	}
	return reader.LastError
}
