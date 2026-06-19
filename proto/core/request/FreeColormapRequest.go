package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type FreeColormapRequest struct {
	Colormap base.COLORMAP
}

func (r *FreeColormapRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.FreeColormap)
	writer.WriteXID(r.Colormap)
	return nil
}
