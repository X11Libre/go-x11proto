package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

// ChangeKeyboardControl value-mask bits.
const (
	KB_MASK_KEY_CLICK_PERCENT = 0x0001
	KB_MASK_BELL_PERCENT      = 0x0002
	KB_MASK_BELL_PITCH        = 0x0004
	KB_MASK_BELL_DURATION     = 0x0008
	KB_MASK_LED               = 0x0010
	KB_MASK_LED_MODE          = 0x0020
	KB_MASK_KEY               = 0x0040
	KB_MASK_AUTO_REPEAT_MODE  = 0x0080
)

type ChangeKeyboardControlRequest struct {
	ValueMask base.CARD32

	KeyClickPercent base.CARD32
	BellPercent     base.CARD32
	BellPitch       base.CARD32
	BellDuration    base.CARD32
	Led             base.CARD32
	LedMode         base.CARD32
	Key             base.CARD32
	AutoRepeatMode  base.CARD32
}

func (r ChangeKeyboardControlRequest) IsMask(m base.CARD32) bool {
	return (r.ValueMask & m) == m
}

func (r *ChangeKeyboardControlRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangeKeyboardControl)
	writer.WriteCARD32(r.ValueMask)
	if r.IsMask(KB_MASK_KEY_CLICK_PERCENT) {
		writer.WriteCARD32(r.KeyClickPercent)
	}
	if r.IsMask(KB_MASK_BELL_PERCENT) {
		writer.WriteCARD32(r.BellPercent)
	}
	if r.IsMask(KB_MASK_BELL_PITCH) {
		writer.WriteCARD32(r.BellPitch)
	}
	if r.IsMask(KB_MASK_BELL_DURATION) {
		writer.WriteCARD32(r.BellDuration)
	}
	if r.IsMask(KB_MASK_LED) {
		writer.WriteCARD32(r.Led)
	}
	if r.IsMask(KB_MASK_LED_MODE) {
		writer.WriteCARD32(r.LedMode)
	}
	if r.IsMask(KB_MASK_KEY) {
		writer.WriteCARD32(r.Key)
	}
	if r.IsMask(KB_MASK_AUTO_REPEAT_MODE) {
		writer.WriteCARD32(r.AutoRepeatMode)
	}
	return nil
}
