package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type SetFontPathRequest struct {
	Path []string
}

func (r *SetFontPathRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetFontPath)
	writer.WriteCARD16(base.CARD16(len(r.Path)))
	writer.WriteCARD16(0) // unused
	for _, s := range r.Path {
		writer.WriteCARD8(base.CARD8(len(s)))
		writer.WriteBytes([]byte(s))
	}
	writer.Pad()
	return nil
}
