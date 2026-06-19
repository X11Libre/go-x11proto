package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type FreeColorsRequest struct {
	Colormap  base.COLORMAP
	PlaneMask base.CARD32
	Pixels    []base.CARD32
}

func (r *FreeColorsRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.FreeColors)
	writer.WriteXID(r.Colormap)
	writer.WriteCARD32(r.PlaneMask)
	for _, p := range r.Pixels {
		writer.WriteCARD32(p)
	}
	return nil
}
