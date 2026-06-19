package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// NoOperationRequest does nothing on the server. The optional Pad units let it
// be used to pad/align the request stream (the body is ignored by the server).
type NoOperationRequest struct {
	Pad base.CARD16 // number of extra 4-byte units of padding
}

func (r *NoOperationRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.NoOperation)
	for i := base.CARD16(0); i < r.Pad; i++ {
		writer.WriteCARD32(0)
	}
	return nil
}
