package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// GetInputFocusRequest has a reply; it is commonly sent (and waited on) purely
// as a round-trip barrier to flush preceding requests.
type GetInputFocusRequest struct{}

func (r *GetInputFocusRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.GetInputFocus)
	return nil
}
