package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type StoreNamedColorRequest struct {
	Flags    base.CARD8
	Colormap base.COLORMAP
	Pixel    base.CARD32
	Name     string
}

func (r *StoreNamedColorRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.StoreNamedColor)
	writer.SetParam0(r.Flags)
	writer.WriteXID(r.Colormap)
	writer.WriteCARD32(r.Pixel)
	writer.WriteCARD16(base.CARD16(len(r.Name)))
	writer.WriteCARD16(0) // unused
	writer.WriteBytes([]byte(r.Name))
	writer.Pad()
	return nil
}
