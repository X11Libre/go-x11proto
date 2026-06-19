package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type QueryColorsRequest struct {
	Colormap base.COLORMAP
	Pixels   []base.CARD32
}

func (r *QueryColorsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.QueryColors)
	writer.WriteXID(r.Colormap)
	for _, p := range r.Pixels {
		writer.WriteCARD32(p)
	}
	return nil
}

// Rgb is an RGB triple as returned by QueryColors.
type Rgb struct {
	Red   base.CARD16
	Green base.CARD16
	Blue  base.CARD16
}

type QueryColorsReply struct {
	Colors []Rgb
}

func (reply *QueryColorsReply) Parse(reader base.ReplyReader) error {
	n := int(reader.CARD16())
	reader.ReadBytes(22) // unused
	reply.Colors = make([]Rgb, 0, n)
	for i := 0; i < n; i++ {
		c := Rgb{Red: reader.CARD16(), Green: reader.CARD16(), Blue: reader.CARD16()}
		reader.ReadBytes(2) // unused
		reply.Colors = append(reply.Colors, c)
	}
	return reader.LastError
}
