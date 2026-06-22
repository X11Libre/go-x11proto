package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

func TestMapWindowEncode(t *testing.T) {
	checkEncode(t, &MapWindowRequest{Window: 0x10}, req(8, 0, u32(0x10)))
}

func TestUnmapWindowEncode(t *testing.T) {
	checkEncode(t, &UnmapWindowRequest{Window: 0x10}, req(10, 0, u32(0x10)))
}

func TestDestroyWindowEncode(t *testing.T) {
	checkEncode(t, &DestroyWindowRequest{Window: 0x10}, req(4, 0, u32(0x10)))
}

func TestFreePixmapEncode(t *testing.T) {
	checkEncode(t, &FreePixmapRequest{Pixmap: 0x10}, req(54, 0, u32(0x10)))
}

func TestGetInputFocusEncode(t *testing.T) {
	checkEncode(t, &GetInputFocusRequest{}, req(43, 0, nil))
}

func TestCreatePixmapEncode(t *testing.T) {
	checkEncode(t, &CreatePixmapRequest{Depth: 24, Pid: 0x10, Drawable: 0x20, Width: 100, Height: 50},
		req(53, 24, cat(u32(0x10), u32(0x20), u16(100), u16(50))))
}

func TestOpenFontEncode(t *testing.T) {
	checkEncode(t, &OpenFontRequest{FontID: 0x10, Name: "fixed"},
		req(45, 0, cat(u32(0x10), u16(5), u16(0), []byte("fixed"))))
}

func TestClearAreaEncode(t *testing.T) {
	checkEncode(t, &ClearAreaRequest{Exposures: true, Window: 0x10, X: 1, Y: 2, Width: 3, Height: 4},
		req(61, 1, cat(u32(0x10), u16(1), u16(2), u16(3), u16(4))))
}

func TestCopyAreaEncode(t *testing.T) {
	checkEncode(t, &CopyAreaRequest{SrcDrawable: 0x10, DstDrawable: 0x20, GC: 0x30,
		SrcX: 1, SrcY: 2, DstX: 3, DstY: 4, Width: 5, Height: 6},
		req(62, 0, cat(u32(0x10), u32(0x20), u32(0x30), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6))))
}

func TestCreateGCEncode(t *testing.T) {
	checkEncode(t, &CreateGCRequest{Gcid: 0x10, Drawable: 0x20, ValueMask: GC_MASK_FOREGROUND, Foreground: 0xAA},
		req(55, 0, cat(u32(0x10), u32(0x20), u32(0x04), u32(0xAA))))
}

func TestCreateWindowEncode(t *testing.T) {
	checkEncode(t, &CreateWindowRequest{Depth: 24, Wid: 0x10, Parent: 0x20, X: 1, Y: 2, Width: 3, Height: 4,
		Border: 0, Class: WindowClass_InputOutput, Visual: 0x30, ValueMask: CW_BACKGROUND_PIXEL, BackPixel: 0xAA},
		req(1, 24, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4), u16(0), u16(1), u32(0x30), u32(0x02), u32(0xAA))))
}

func TestChangeWindowAttributesEncode(t *testing.T) {
	checkEncode(t, &ChangeWindowAttributesRequest{Window: 0x10, ValueMask: CW_BACKGROUND_PIXEL, BackPixel: 0xAA},
		req(2, 0, cat(u32(0x10), u32(0x02), u32(0xAA))))
}

func TestConfigureWindowEncode(t *testing.T) {
	checkEncode(t, &ConfigureWindowRequest{Window: 0x10, ValueMask: CONFIG_WINDOW_X | CONFIG_WINDOW_Y, X: 5, Y: 6},
		req(12, 0, cat(u32(0x10), u16(0x03), u16(0), u32(5), u32(6))))
}

func TestPolyFillRectEncode(t *testing.T) {
	checkEncode(t, &PolyFillRectRequest{Drawable: 0x10, Gc: 0x20,
		Rects: []base.Rectangle{{X: 1, Y: 2, Width: 3, Height: 4}}},
		req(70, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4))))
}

func TestPutImageEncode(t *testing.T) {
	checkEncode(t, &PutImageRequest{Format: 2, Drawable: 0x10, Gc: 0x20, Width: 2, Height: 1,
		DstX: 3, DstY: 4, LeftPad: 0, Depth: 24, Data: []byte{1, 2, 3, 4}},
		req(72, 2, cat(u32(0x10), u32(0x20), u16(2), u16(1), u16(3), u16(4), u8(0), u8(24), u8(0), u8(0), []byte{1, 2, 3, 4})))
}

func TestPutText8Encode(t *testing.T) {
	checkEncode(t, &PutText8Request{Drawable: 0x10, Gc: 0x20, X: 1, Y: 2, Text: "Hi"},
		req(74, 2, cat(u32(0x10), u32(0x20), u16(1), u16(2), u8(2), u8(0), []byte("Hi"))))
}

func TestChangePropertyEncode(t *testing.T) {
	checkEncode(t, &ChangePropertyRequest{Mode: 0, Window: 0x10, Property: 0xAB, Type: 0xCD,
		Format: 8, Data8: []base.CARD8{1, 2, 3}},
		req(18, 0, cat(u32(0x10), u32(0xAB), u32(0xCD), u8(8), u8(0), u8(0), u8(0), u32(3), u8(1), u8(2), u8(3), u8(0))))
}

func TestSendEventEncode(t *testing.T) {
	var ev [32]byte
	ev[0] = 33 // ClientMessage
	ev[1] = 32
	checkEncode(t, &SendEventRequest{Propagate: false, Destination: 0x10, EventMask: 0xFF, Event: ev},
		req(25, 0, cat(u32(0x10), u32(0xFF), ev[:])))
}

func TestInternAtomEncode(t *testing.T) {
	checkEncode(t, &InternAtomRequest{Name: "WM_NAME"},
		req(16, 0, cat(u16(7), u16(0), []byte("WM_NAME"))))
}

func TestInternAtomReply(t *testing.T) {
	got := &InternAtomReply{}
	if err := got.Parse(makeReply(0, 0, cat(u32(0x123), make([]byte, 20)))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &InternAtomReply{Atom: 0x123})
}

func TestQueryExtensionEncode(t *testing.T) {
	checkEncode(t, &QueryExtensionRequest{Name: "SHAPE"},
		req(98, 0, cat(u16(5), u16(0), []byte("SHAPE"))))
}

func TestQueryExtensionReply(t *testing.T) {
	got := &QueryExtensionReply{}
	// reply payload (offset 8): present, major-opcode, first-event, first-error;
	// data0 (byte 1) is unused.
	if err := got.Parse(makeReply(0, 0, cat(u8(1), u8(128), u8(64), u8(65), make([]byte, 20)))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryExtensionReply{Present: true, MajorOpcode: 128, FirstEvent: 64, FirstError: 65})
}

func TestListExtensionsEncode(t *testing.T) {
	checkEncode(t, &ListExtensionsRequest{}, req(99, 0, nil))
}

func TestListExtensionsReply(t *testing.T) {
	tail := cat(make([]byte, 24), u8(5), []byte("SHAPE"), u8(5), []byte("XTEST"))
	got := &ListExtensionsReply{}
	if err := got.Parse(makeReply(2, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &ListExtensionsReply{Names: []string{"SHAPE", "XTEST"}})
}
