package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// QueryBestSize class (param0).
const (
	BestSizeCursor  = 0
	BestSizeTile    = 1
	BestSizeStipple = 2
)

type QueryBestSizeRequest struct {
	Class    base.CARD8
	Drawable base.DRAWABLE
	Width    base.CARD16
	Height   base.CARD16
}

func (r *QueryBestSizeRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryBestSize)
	writer.SetParam0(r.Class)
	writer.WriteXID(r.Drawable)
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	return nil
}

type QueryBestSizeReply struct {
	Width  base.CARD16
	Height base.CARD16
}

func (reply *QueryBestSizeReply) Parse(reader base.ReplyReader) error {
	reply.Width = reader.CARD16()
	reply.Height = reader.CARD16()
	return reader.LastError
}
