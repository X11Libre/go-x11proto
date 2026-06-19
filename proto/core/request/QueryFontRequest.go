package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// QueryFont takes a FONTABLE (a FONT or a GC).
type QueryFontRequest struct {
	Font base.FONT
}

func (r *QueryFontRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryFont)
	writer.WriteXID(r.Font)
	return nil
}

type QueryFontReply struct {
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
	Properties     []base.FontProp
	CharInfos      []base.CharInfo
}

func readCharInfo(reader *base.ReplyReader) base.CharInfo {
	return base.CharInfo{
		LeftSideBearing:  reader.INT16(),
		RightSideBearing: reader.INT16(),
		CharacterWidth:   reader.INT16(),
		Ascent:           reader.INT16(),
		Descent:          reader.INT16(),
		Attributes:       reader.CARD16(),
	}
}

func (reply *QueryFontReply) Parse(reader base.ReplyReader) error {
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
	nChars := int(reader.CARD32())
	reply.Properties = make([]base.FontProp, 0, nProps)
	for i := 0; i < nProps; i++ {
		reply.Properties = append(reply.Properties, base.FontProp{
			Name:  base.ATOM(reader.CARD32()),
			Value: reader.CARD32(),
		})
	}
	reply.CharInfos = make([]base.CharInfo, 0, nChars)
	for i := 0; i < nChars; i++ {
		reply.CharInfos = append(reply.CharInfos, readCharInfo(&reader))
	}
	return reader.LastError
}
