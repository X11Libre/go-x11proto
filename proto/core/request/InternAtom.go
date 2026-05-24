package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type InternAtomRequest struct {
	OnlyIfExist bool
	Name        string
}

func (r *InternAtomRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.InternAtom)

	if r.OnlyIfExist {
		writer.SetParam0(1)
	}

	writer.WriteCARD16(base.CARD16(len(r.Name)))
	writer.WriteCARD16(0) // pad0
	writer.WriteBytes([]byte(r.Name))
	return nil
}

type InternAtomReply struct {
	Atom base.ATOM
}

func (reply *InternAtomReply) Parse(reader base.ReplyReader) error {
	reply.Atom = base.ATOM(reader.CARD32())
	return reader.LastError
}
