package request

import (
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/opcode"
)

const (
	WindowClass_CopyFromParent = 0
	WindowClass_InputOutput    = 1
	WindowClass_InputOnly      = 2
)

const (
	CW_BACKGROUND_PIXMAP = 0x0001
	CW_BACKGROUND_PIXEL  = 0x0002
	CW_BORDER_PIXMAP     = 0x0004
	CW_BORDER_PIXEL      = 0x0008
	CW_BIT_GRAVITY       = 0x0010
	CW_WIN_GRAVITY       = 0x0020
	CW_BACKING_STORE     = 0x0040
	CW_BACKING_PLANES    = 0x0080
	CW_BACKING_PIXEL     = 0x0100
	CW_OVERRIDE_REDIRECT = 0x0200
	CW_SAVE_UNDER        = 0x0400
	CW_EVENT_MASK        = 0x0800
	CW_DO_NOT_PROPAGATE  = 0x1000
	CW_COLORMAP          = 0x2000
	CW_CURSOR            = 0x4000
)

type CreateWindowRequest struct {
	Depth     base.CARD8
	Wid       base.WINDOW
	Parent    base.WINDOW
	X         base.INT16
	Y         base.INT16
	Width     base.CARD16
	Height    base.CARD16
	Border    base.CARD16
	Class     base.CARD16
	Visual    base.VISUAL
	ValueMask base.CARD32

	// optional values
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

func (r CreateWindowRequest) IsMask(m base.CARD32) bool {
	return (r.ValueMask & m) == m
}

func (r *CreateWindowRequest) WriteInto(writer *base.RequestWriter) error {
	writer.SetOpcode(opcode.CreateWindow)
	writer.SetParam0(r.Depth)
	writer.WriteXID(r.Wid)
	writer.WriteXID(r.Parent)
	writer.WriteINT16(r.X)
	writer.WriteINT16(r.Y)
	writer.WriteCARD16(r.Width)
	writer.WriteCARD16(r.Height)
	writer.WriteCARD16(r.Border)
	writer.WriteCARD16(r.Class)
	writer.WriteXID(r.Visual)
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
