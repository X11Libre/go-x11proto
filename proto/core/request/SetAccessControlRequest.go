package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// SetAccessControl mode (param0).
const (
	AccessControlDisable = 0
	AccessControlEnable  = 1
)

type SetAccessControlRequest struct {
	Mode base.CARD8
}

func (r *SetAccessControlRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.SetAccessControl)
	writer.SetParam0(r.Mode)
	return nil
}
