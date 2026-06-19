package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// ForceScreenSaver mode (param0).
const (
	ScreenSaverReset    = 0
	ScreenSaverActivate = 1
)

type ForceScreenSaverRequest struct {
	Mode base.CARD8
}

func (r *ForceScreenSaverRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ForceScreenSaver)
	writer.SetParam0(r.Mode)
	return nil
}
