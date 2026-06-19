package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// GetImage format (the param0 byte).
const (
	ImageFormatXYPixmap = 1
	ImageFormatZPixmap  = 2
)

type GetImageRequest struct {
	Format    base.CARD8
	Drawable  base.DRAWABLE
	X         base.INT16
	Y         base.INT16
	Width     base.CARD16
	Height    base.CARD16
	PlaneMask base.CARD32
}

func (r *GetImageRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetImage)
	writer.SetParam0(r.Format)
	writer.WriteXID(r.Drawable)
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	writer.WriteCARD32(r.PlaneMask)
	return nil
}

type GetImageReply struct {
	Depth  base.CARD8
	Visual base.VISUAL
	Data   []byte
}

func (reply *GetImageReply) Parse(reader base.ReplyReader) error {
	reply.Depth = reader.Data0 // depth carried in the reply header byte
	reply.Visual = base.VISUAL(reader.CARD32())
	reader.ReadBytes(20) // unused
	reply.Data = reader.ReadBytes(uint(reader.Length) * 4)
	return reader.LastError
}
