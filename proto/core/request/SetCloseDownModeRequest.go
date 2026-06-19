package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// SetCloseDownMode mode (param0).
const (
	CloseDownDestroy         = 0
	CloseDownRetainPermanent = 1
	CloseDownRetainTemporary = 2
)

type SetCloseDownModeRequest struct {
	Mode base.CARD8
}

func (r *SetCloseDownModeRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetCloseDownMode)
	writer.SetParam0(r.Mode)
	return nil
}
