package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ListFontsWithInfoRequest struct {
	MaxNames base.CARD16
	Pattern  string
}

func (r *ListFontsWithInfoRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ListFontsWithInfo)
	writer.WriteCARD16(r.MaxNames)
	writer.WriteCARD16(base.CARD16(len(r.Pattern)))
	writer.WriteBytes([]byte(r.Pattern))
	writer.Pad()
	return nil
}

// ListFontsWithInfoReply is one reply in the series. The server sends one per
// matching font followed by a terminating reply with LastReply set. Parsing the
// full series needs multi-reply support in the connection layer (see the RPC).
type ListFontsWithInfoReply struct {
	LastReply      bool
	MinBounds      base.CharInfo
	MaxBounds      base.CharInfo
	MinCharOrByte2 base.CARD16
	MaxCharOrByte2 base.CARD16
	DefaultChar    base.CARD16
	DrawDirection  base.CARD8
	MinByte1       base.CARD8
	MaxByte1       base.CARD8
	AllCharsExist  bool
	FontAscent     base.INT16
	FontDescent    base.INT16
	RepliesHint    base.CARD32
	Properties     []base.FontProp
	Name           string
}

func (reply *ListFontsWithInfoReply) Parse(reader base.ReplyReader) error {
	n := int(reader.Data0) // name length; 0 marks the terminating reply
	reply.LastReply = n == 0
	reply.MinBounds = readCharInfo(&reader)
	reader.ReadBytes(4) // unused
	reply.MaxBounds = readCharInfo(&reader)
	reader.ReadBytes(4) // unused
	reply.MinCharOrByte2 = reader.CARD16()
	reply.MaxCharOrByte2 = reader.CARD16()
	reply.DefaultChar = reader.CARD16()
	nProps := int(reader.CARD16())
	reply.DrawDirection = reader.CARD8()
	reply.MinByte1 = reader.CARD8()
	reply.MaxByte1 = reader.CARD8()
	reply.AllCharsExist = reader.Bool()
	reply.FontAscent = reader.INT16()
	reply.FontDescent = reader.INT16()
	reply.RepliesHint = reader.CARD32()
	reply.Properties = make([]base.FontProp, 0, nProps)
	for i := 0; i < nProps; i++ {
		reply.Properties = append(reply.Properties, base.FontProp{
			Name:  base.ATOM(reader.CARD32()),
			Value: reader.CARD32(),
		})
	}
	reply.Name = string(reader.ReadBytes(uint(n)))
	return reader.LastError
}
