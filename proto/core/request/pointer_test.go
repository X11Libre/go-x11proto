package request

import "testing"

func TestQueryPointerEncode(t *testing.T) {
	checkEncode(t, &QueryPointerRequest{Window: 0x10}, req(38, 0, u32(0x10)))
}

func TestQueryPointerReply(t *testing.T) {
	tail := cat(u32(0x1), u32(0x2), u16(10), u16(20), u16(30), u16(40), u16(0x50), make([]byte, 6))
	got := &QueryPointerReply{}
	if err := got.Parse(makeReply(1, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryPointerReply{SameScreen: true, Root: 0x1, Child: 0x2,
		RootX: 10, RootY: 20, WinX: 30, WinY: 40, Mask: 0x50})
}

func TestGetMotionEventsEncode(t *testing.T) {
	checkEncode(t, &GetMotionEventsRequest{Window: 0x10, Start: 1, Stop: 2},
		req(39, 0, cat(u32(0x10), u32(1), u32(2))))
}

func TestGetMotionEventsReply(t *testing.T) {
	tail := cat(u32(1), make([]byte, 20), u32(0x100), u16(5), u16(6))
	got := &GetMotionEventsReply{}
	if err := got.Parse(makeReply(0, 2, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetMotionEventsReply{Events: []TimeCoord{{Time: 0x100, X: 5, Y: 6}}})
}

func TestTranslateCoordinatesEncode(t *testing.T) {
	checkEncode(t, &TranslateCoordinatesRequest{SrcWindow: 0x10, DstWindow: 0x20, SrcX: 1, SrcY: 2},
		req(40, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2))))
}

func TestTranslateCoordinatesReply(t *testing.T) {
	tail := cat(u32(0x5), u16(7), u16(8), make([]byte, 16))
	got := &TranslateCoordinatesReply{}
	if err := got.Parse(makeReply(1, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &TranslateCoordinatesReply{SameScreen: true, Child: 0x5, DstX: 7, DstY: 8})
}

func TestWarpPointerEncode(t *testing.T) {
	checkEncode(t, &WarpPointerRequest{SrcWindow: 0x10, DstWindow: 0x20, SrcX: 1, SrcY: 2,
		SrcWidth: 3, SrcHeight: 4, DstX: 5, DstY: 6},
		req(41, 0, cat(u32(0x10), u32(0x20), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6))))
}

func TestSetInputFocusEncode(t *testing.T) {
	checkEncode(t, &SetInputFocusRequest{RevertTo: RevertToPointerRoot, Focus: 0x10, Time: 0x123},
		req(42, 1, cat(u32(0x10), u32(0x123))))
}

func TestQueryKeymapEncode(t *testing.T) {
	checkEncode(t, &QueryKeymapRequest{}, req(44, 0, nil))
}

func TestQueryKeymapReply(t *testing.T) {
	tail := make([]byte, 32)
	for i := range tail {
		tail[i] = byte(i)
	}
	got := &QueryKeymapReply{}
	if err := got.Parse(makeReply(0, 8, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryKeymapReply{Keys: tail})
}
