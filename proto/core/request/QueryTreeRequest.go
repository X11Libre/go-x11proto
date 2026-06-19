package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type QueryTreeRequest struct {
	Window base.WINDOW
}

func (r *QueryTreeRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryTree)
	writer.WriteXID(r.Window)
	return nil
}

type QueryTreeReply struct {
	Root     base.WINDOW
	Parent   base.WINDOW
	Children []base.WINDOW
}

func (reply *QueryTreeReply) Parse(reader base.ReplyReader) error {
	reply.Root = reader.XID()
	reply.Parent = reader.XID()
	n := int(reader.CARD16())
	reader.ReadBytes(14) // unused
	reply.Children = make([]base.WINDOW, 0, n)
	for i := 0; i < n; i++ {
		reply.Children = append(reply.Children, base.WINDOW(reader.XID()))
	}
	return reader.LastError
}
