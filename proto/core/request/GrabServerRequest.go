package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type GrabServerRequest struct{}

func (r *GrabServerRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GrabServer)
	return nil
}
