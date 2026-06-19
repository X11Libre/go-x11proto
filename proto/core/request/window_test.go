package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

func TestDestroySubwindowsEncode(t *testing.T) {
	checkEncode(t, &DestroySubwindowsRequest{Window: 0x111},
		cat(u8(5), u8(0), u16(2), u32(0x111)))
}

func TestMapSubwindowsEncode(t *testing.T) {
	checkEncode(t, &MapSubwindowsRequest{Window: 0x222},
		cat(u8(9), u8(0), u16(2), u32(0x222)))
}

func TestUnmapSubwindowsEncode(t *testing.T) {
	checkEncode(t, &UnmapSubwindowsRequest{Window: 0x333},
		cat(u8(11), u8(0), u16(2), u32(0x333)))
}

func TestCirculateWindowEncode(t *testing.T) {
	checkEncode(t, &CirculateWindowRequest{Direction: CirculateLowerHighest, Window: 0x444},
		cat(u8(13), u8(1), u16(2), u32(0x444)))
}

func TestChangeSaveSetEncode(t *testing.T) {
	checkEncode(t, &ChangeSaveSetRequest{Mode: SaveSetDelete, Window: 0x555},
		cat(u8(6), u8(1), u16(2), u32(0x555)))
}

func TestReparentWindowEncode(t *testing.T) {
	checkEncode(t, &ReparentWindowRequest{Window: 0x111, Parent: 0x222, X: 5, Y: 6},
		cat(u8(7), u8(0), u16(4), u32(0x111), u32(0x222), u16(5), u16(6)))
}

func TestGetGeometryEncode(t *testing.T) {
	checkEncode(t, &GetGeometryRequest{Drawable: 0x111},
		cat(u8(14), u8(0), u16(2), u32(0x111)))
}

func TestGetGeometryReply(t *testing.T) {
	tail := cat(u32(0x33), u16(10), u16(20), u16(640), u16(480), u16(2), make([]byte, 10))
	r := makeReply(24, 0, tail)
	got := &GetGeometryReply{}
	if err := got.Parse(r); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetGeometryReply{
		Depth: 24, Root: 0x33, X: 10, Y: 20, Width: 640, Height: 480, BorderWidth: 2,
	})
}

func TestGetWindowAttributesEncode(t *testing.T) {
	checkEncode(t, &GetWindowAttributesRequest{Window: 0x111},
		cat(u8(3), u8(0), u16(2), u32(0x111)))
}

func TestGetWindowAttributesReply(t *testing.T) {
	tail := cat(
		u32(0x20),  // visual
		u16(1),     // class
		u8(1),      // bit-gravity
		u8(1),      // win-gravity
		u32(0xFF),  // backing-planes
		u32(0xAA),  // backing-pixel
		u8(1),      // save-under
		u8(1),      // map-is-installed
		u8(2),      // map-state
		u8(0),      // override-redirect
		u32(0x44),  // colormap
		u32(0xFFFF),// all-event-masks
		u32(0x8000),// your-event-mask
		u16(0x10),  // do-not-propagate
		u16(0),     // unused
	)
	r := makeReply(1, 3, tail)
	got := &GetWindowAttributesReply{}
	if err := got.Parse(r); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetWindowAttributesReply{
		BackingStore: 1, Visual: 0x20, Class: 1, BitGravity: 1, WinGravity: 1,
		BackingPlanes: 0xFF, BackingPixel: 0xAA, SaveUnder: true, MapIsInstalled: true,
		MapState: 2, OverrideRedirect: false, Colormap: 0x44, AllEventMasks: 0xFFFF,
		YourEventMask: 0x8000, DoNotPropagateMask: 0x10,
	})
}

func TestQueryTreeEncode(t *testing.T) {
	checkEncode(t, &QueryTreeRequest{Window: 0x111},
		cat(u8(15), u8(0), u16(2), u32(0x111)))
}

func TestQueryTreeReply(t *testing.T) {
	tail := cat(u32(1), u32(2), u16(2), make([]byte, 14), u32(0x10), u32(0x11))
	r := makeReply(0, 2, tail)
	got := &QueryTreeReply{}
	if err := got.Parse(r); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryTreeReply{
		Root: 1, Parent: 2, Children: []base.WINDOW{0x10, 0x11},
	})
}
