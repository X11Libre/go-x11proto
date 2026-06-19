package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type QueryPointerRequest struct {
	Window base.WINDOW
}

func (r *QueryPointerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryPointer)
	writer.WriteXID(r.Window)
	return nil
}

type QueryPointerReply struct {
	SameScreen bool
	Root       base.WINDOW
	Child      base.WINDOW
	RootX      base.INT16
	RootY      base.INT16
	WinX       base.INT16
	WinY       base.INT16
	Mask       base.CARD16
}

func (reply *QueryPointerReply) Parse(reader base.ReplyReader) error {
	reply.SameScreen = reader.Data0 != 0
	reply.Root = reader.XID()
	reply.Child = reader.XID()
	reply.RootX = reader.INT16()
	reply.RootY = reader.INT16()
	reply.WinX = reader.INT16()
	reply.WinY = reader.INT16()
	reply.Mask = reader.CARD16()
	return reader.LastError
}
