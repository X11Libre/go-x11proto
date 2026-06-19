package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// CreateColormap alloc (the param0 byte).
const (
	ColormapAllocNone = 0
	ColormapAllocAll  = 1
)

type CreateColormapRequest struct {
	Alloc  base.CARD8
	Mid    base.COLORMAP
	Window base.WINDOW
	Visual base.VISUAL
}

func (r *CreateColormapRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CreateColormap)
	writer.SetParam0(r.Alloc)
	writer.WriteXID(r.Mid)
	writer.WriteXID(r.Window)
	writer.WriteXID(r.Visual)
	return nil
}
