package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// AllTemporary may be used as the resource to kill all RetainTemporary clients.
const AllTemporary = 0

type KillClientRequest struct {
	Resource base.CARD32
}

func (r *KillClientRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.KillClient)
	writer.WriteCARD32(r.Resource)
	return nil
}
