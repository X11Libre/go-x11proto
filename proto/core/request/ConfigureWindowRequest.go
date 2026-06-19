package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

const (
	CONFIG_WINDOW_X            = 0x0001
	CONFIG_WINDOW_Y            = 0x0002
	CONFIG_WINDOW_WIDTH        = 0x0004
	CONFIG_WINDOW_HEIGHT       = 0x0008
	CONFIG_WINDOW_BORDER_WIDTH = 0x0010
	CONFIG_WINDOW_SIBLING      = 0x0020
	CONFIG_WINDOW_STACK_MODE   = 0x0040
)

// Stack mode (the value for CONFIG_WINDOW_STACK_MODE).
const (
	StackModeAbove    = 0
	StackModeBelow    = 1
	StackModeTopIf    = 2
	StackModeBottomIf = 3
	StackModeOpposite = 4
)

type ConfigureWindowRequest struct {
	Window      base.WINDOW
	ValueMask   base.CARD16
	X           base.INT16
	Y           base.INT16
	Width       base.CARD16
	Height      base.CARD16
	BorderWidth base.CARD16
	Sibling     base.WINDOW
	StackMode   base.CARD32
}

func (r ConfigureWindowRequest) IsMask(m base.CARD16) bool {
	return (r.ValueMask & m) == m
}

func (r *ConfigureWindowRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ConfigureWindow)
	writer.WriteXID(base.XID(r.Window))
	writer.WriteCARD16(r.ValueMask)
	writer.WriteCARD16(0) // unused
	// each value-list entry is 4 bytes
	if r.IsMask(CONFIG_WINDOW_X) {
		writer.WriteCARD32(base.CARD32(uint16(r.X)))
	}
	if r.IsMask(CONFIG_WINDOW_Y) {
		writer.WriteCARD32(base.CARD32(uint16(r.Y)))
	}
	if r.IsMask(CONFIG_WINDOW_WIDTH) {
		writer.WriteCARD32(base.CARD32(r.Width))
	}
	if r.IsMask(CONFIG_WINDOW_HEIGHT) {
		writer.WriteCARD32(base.CARD32(r.Height))
	}
	if r.IsMask(CONFIG_WINDOW_BORDER_WIDTH) {
		writer.WriteCARD32(base.CARD32(r.BorderWidth))
	}
	if r.IsMask(CONFIG_WINDOW_SIBLING) {
		writer.WriteXID(base.XID(r.Sibling))
	}
	if r.IsMask(CONFIG_WINDOW_STACK_MODE) {
		writer.WriteCARD32(r.StackMode)
	}
	return nil
}
