package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ListFontsRequest struct {
	MaxNames base.CARD16
	Pattern  string
}

func (r *ListFontsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ListFonts)
	writer.WriteCARD16(r.MaxNames)
	writer.WriteCARD16(base.CARD16(len(r.Pattern)))
	writer.WriteBytes([]byte(r.Pattern))
	writer.Pad()
	return nil
}

type ListFontsReply struct {
	Names []string
}

// parseStrList reads a count-prefixed list of length-prefixed strings (STR).
func parseStrList(reader *base.ReplyReader, count int) []string {
	names := make([]string, 0, count)
	for i := 0; i < count; i++ {
		l := uint(reader.CARD8())
		names = append(names, string(reader.ReadBytes(l)))
	}
	return names
}

func (reply *ListFontsReply) Parse(reader base.ReplyReader) error {
	n := int(reader.CARD16())
	reader.ReadBytes(22) // unused
	reply.Names = parseStrList(&reader, n)
	return reader.LastError
}
