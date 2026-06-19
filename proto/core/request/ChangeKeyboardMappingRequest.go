package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// ChangeKeyboardMapping sets KeysymsPerKeycode keysyms for KeycodeCount keycodes
// starting at FirstKeycode. Keysyms is laid out row-major (count x per).
type ChangeKeyboardMappingRequest struct {
	FirstKeycode      base.CARD8
	KeysymsPerKeycode base.CARD8
	KeycodeCount      base.CARD8
	Keysyms           []base.CARD32
}

func (r *ChangeKeyboardMappingRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangeKeyboardMap)
	writer.SetParam0(r.KeycodeCount)
	writer.WriteCARD8(r.FirstKeycode)
	writer.WriteCARD8(r.KeysymsPerKeycode)
	writer.WriteCARD16(0) // unused
	for _, ks := range r.Keysyms {
		writer.WriteCARD32(ks)
	}
	return nil
}
