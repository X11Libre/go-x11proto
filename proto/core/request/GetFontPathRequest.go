package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetFontPathRequest struct{}

func (r *GetFontPathRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetFontPath)
	return nil
}

type GetFontPathReply struct {
	Path []string
}

func (reply *GetFontPathReply) Parse(reader base.ReplyReader) error {
	n := int(reader.CARD16())
	reader.ReadBytes(22) // unused
	reply.Path = parseStrList(&reader, n)
	return reader.LastError
}
