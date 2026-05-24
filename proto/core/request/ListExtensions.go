package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ListExtensionsRequest struct {
}

func (r *ListExtensionsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ListExtensions)
	return nil
}

type ListExtensionsReply struct {
	Names []string
}

func (reply *ListExtensionsReply) Parse(reader base.ReplyReader) error {
	count := int(reader.Data0)
	for i := 0; i < 6; i++ {
		reader.CARD32() // skip 24 bytes
	}
	list := []string{}

	for i := 0; i < count; i++ {
		list = append(list, reader.ReadString())
	}

	reply.Names = list
	return reader.LastError
}
