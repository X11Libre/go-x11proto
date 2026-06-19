package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type QueryKeymapRequest struct{}

func (r *QueryKeymapRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryKeymap)
	return nil
}

type QueryKeymapReply struct {
	Keys []byte // 32-byte (256-bit) keycode bitmap
}

func (reply *QueryKeymapReply) Parse(reader base.ReplyReader) error {
	reply.Keys = reader.ReadBytes(32)
	return reader.LastError
}
