package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

func TestChangeGCEncode(t *testing.T) {
	checkEncode(t, &ChangeGCRequest{
		Gc: 0x10, ValueMask: GC_MASK_FOREGROUND | GC_MASK_BACKGROUND,
		Foreground: 0xAA, Background: 0xBB,
	}, req(56, 0, cat(u32(0x10), u32(0x0C), u32(0xAA), u32(0xBB))))
}

func TestCopyGCEncode(t *testing.T) {
	checkEncode(t, &CopyGCRequest{SrcGC: 0x10, DstGC: 0x20, ValueMask: 0x04},
		req(57, 0, cat(u32(0x10), u32(0x20), u32(0x04))))
}

func TestFreeGCEncode(t *testing.T) {
	checkEncode(t, &FreeGCRequest{Gc: 0x10}, req(60, 0, u32(0x10)))
}

func TestSetDashesEncode(t *testing.T) {
	checkEncode(t, &SetDashesRequest{Gc: 0x10, DashOffset: 1, Dashes: []base.CARD8{4, 2, 4}},
		req(58, 0, cat(u32(0x10), u16(1), u16(3), []byte{4, 2, 4})))
}

func TestSetClipRectanglesEncode(t *testing.T) {
	checkEncode(t, &SetClipRectanglesRequest{
		Ordering: ClipOrderingYSorted, Gc: 0x10, ClipXOrigin: 1, ClipYOrigin: 2,
		Rectangles: []base.Rectangle{{X: 3, Y: 4, Width: 5, Height: 6}},
	}, req(59, 1, cat(u32(0x10), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6))))
}
