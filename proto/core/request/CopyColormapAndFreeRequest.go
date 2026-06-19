package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type CopyColormapAndFreeRequest struct {
	Mid    base.COLORMAP
	SrcMap base.COLORMAP
}

func (r *CopyColormapAndFreeRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CopyColormap)
	writer.WriteXID(r.Mid)
	writer.WriteXID(r.SrcMap)
	return nil
}
