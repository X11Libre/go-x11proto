package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetAtomNameRequest struct {
	Atom base.ATOM
}

func (r *GetAtomNameRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetAtomName)
	writer.WriteATOM(r.Atom)
	return nil
}

type GetAtomNameReply struct {
	Name string
}

func (reply *GetAtomNameReply) Parse(reader base.ReplyReader) error {
	n := reader.CARD16()
	reader.ReadBytes(22) // unused
	reply.Name = string(reader.ReadBytes(uint(n)))
	return reader.LastError
}
