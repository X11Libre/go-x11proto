package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

type ChangeWindowAttributesRequest struct {
	Window    base.WINDOW
	ValueMask base.CARD32

	BackPixmap       base.XID
	BackPixel        base.CARD32
	BorderPixmap     base.XID
	BorderPixel      base.CARD32
	BitGravity       base.CARD32
	WinGravity       base.CARD32
	BackingStore     base.CARD32
	BackingPlanes    base.CARD32
	BackingPixel     base.CARD32
	OverrideRedirect base.CARD32
	SaveUnder        base.CARD32
	EventMask        base.CARD32
	DoNotPropagate   base.CARD32
	Cursor           base.CARD32
}

func (r ChangeWindowAttributesRequest) IsMask(m base.CARD32) bool {
	return (r.ValueMask & m) == m
}

func (r *ChangeWindowAttributesRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.ChangeWindowAttr)
	writer.WriteXID(base.XID(r.Window))
	writer.WriteCARD32(r.ValueMask)

	if r.IsMask(CW_BACKGROUND_PIXMAP) {
		writer.WriteXID(r.BackPixmap)
	}
	if r.IsMask(CW_BACKGROUND_PIXEL) {
		writer.WriteCARD32(r.BackPixel)
	}
	if r.IsMask(CW_BORDER_PIXMAP) {
		writer.WriteXID(r.BorderPixmap)
	}
	if r.IsMask(CW_BORDER_PIXEL) {
		writer.WriteCARD32(r.BorderPixel)
	}
	if r.IsMask(CW_BIT_GRAVITY) {
		writer.WriteCARD32(r.BitGravity)
	}
	if r.IsMask(CW_WIN_GRAVITY) {
		writer.WriteCARD32(r.WinGravity)
	}
	if r.IsMask(CW_BACKING_STORE) {
		writer.WriteCARD32(r.BackingStore)
	}
	if r.IsMask(CW_BACKING_PLANES) {
		writer.WriteCARD32(r.BackingPlanes)
	}
	if r.IsMask(CW_BACKING_PIXEL) {
		writer.WriteCARD32(r.BackingPixel)
	}
	if r.IsMask(CW_OVERRIDE_REDIRECT) {
		writer.WriteCARD32(r.OverrideRedirect)
	}
	if r.IsMask(CW_SAVE_UNDER) {
		writer.WriteCARD32(r.SaveUnder)
	}
	if r.IsMask(CW_EVENT_MASK) {
		writer.WriteCARD32(r.EventMask)
	}
	if r.IsMask(CW_DO_NOT_PROPAGATE) {
		writer.WriteCARD32(r.DoNotPropagate)
	}
	if r.IsMask(CW_CURSOR) {
		writer.WriteCARD32(r.Cursor)
	}
	return nil
}
