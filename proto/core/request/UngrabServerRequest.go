package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type UngrabServerRequest struct{}

func (r *UngrabServerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.UngrabServer)
	return nil
}
