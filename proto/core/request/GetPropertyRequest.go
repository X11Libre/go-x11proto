package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GetPropertyRequest struct {
	Delete     bool
	Window     base.WINDOW
	Property   base.ATOM
	Type       base.ATOM // AnyPropertyType (0) to match any
	LongOffset base.CARD32
	LongLength base.CARD32
}

func (r *GetPropertyRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetProperty)
	writer.SetParam0bool(r.Delete)
	writer.WriteXID(r.Window)
	writer.WriteATOM(r.Property)
	writer.WriteATOM(r.Type)
	writer.WriteCARD32(r.LongOffset)
	writer.WriteCARD32(r.LongLength)
	return nil
}

type GetPropertyReply struct {
	Format     base.CARD8 // 0, 8, 16 or 32
	Type       base.ATOM
	BytesAfter base.CARD32
	ValueLen   base.CARD32 // number of items of size Format
	Value      []byte
}

func (reply *GetPropertyReply) Parse(reader base.ReplyReader) error {
	reply.Format = reader.Data0
	reply.Type = base.ATOM(reader.CARD32())
	reply.BytesAfter = reader.CARD32()
	reply.ValueLen = reader.CARD32()
	reader.ReadBytes(12) // unused
	reply.Value = reader.ReadBytes(uint(reply.ValueLen) * uint(reply.Format) / 8)
	return reader.LastError
}
