package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ListPropertiesRequest struct {
	Window base.WINDOW
}

func (r *ListPropertiesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ListProperties)
	writer.WriteXID(r.Window)
	return nil
}

type ListPropertiesReply struct {
	Atoms []base.ATOM
}

func (reply *ListPropertiesReply) Parse(reader base.ReplyReader) error {
	n := int(reader.CARD16())
	reader.ReadBytes(22) // unused
	reply.Atoms = make([]base.ATOM, 0, n)
	for i := 0; i < n; i++ {
		reply.Atoms = append(reply.Atoms, base.ATOM(reader.CARD32()))
	}
	return reader.LastError
}
