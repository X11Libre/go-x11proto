package request

import "testing"

func TestGrabPointerEncode(t *testing.T) {
	checkEncode(t, &GrabPointerRequest{OwnerEvents: true, GrabWindow: 0x10, EventMask: 0xFF,
		PointerMode: GrabModeAsync, KeyboardMode: GrabModeSync, ConfineTo: 0x20, Cursor: 0x30, Time: 0x123},
		req(26, 1, cat(u32(0x10), u16(0xFF), u8(1), u8(0), u32(0x20), u32(0x30), u32(0x123))))
}

func TestGrabPointerReply(t *testing.T) {
	got := &GrabPointerReply{}
	if err := got.Parse(makeReply(1, 0, make([]byte, 24))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GrabPointerReply{Status: 1})
}

func TestUngrabPointerEncode(t *testing.T) {
	checkEncode(t, &UngrabPointerRequest{Time: 0x123}, req(27, 0, u32(0x123)))
}

func TestGrabButtonEncode(t *testing.T) {
	checkEncode(t, &GrabButtonRequest{OwnerEvents: false, GrabWindow: 0x10, EventMask: 0xFF,
		PointerMode: GrabModeAsync, KeyboardMode: GrabModeSync, ConfineTo: 0x20, Cursor: 0x30, Button: 1, Modifiers: 0x8000},
		req(28, 0, cat(u32(0x10), u16(0xFF), u8(1), u8(0), u32(0x20), u32(0x30), u8(1), u8(0), u16(0x8000))))
}

func TestUngrabButtonEncode(t *testing.T) {
	checkEncode(t, &UngrabButtonRequest{Button: 1, GrabWindow: 0x10, Modifiers: 0x8000},
		req(29, 1, cat(u32(0x10), u16(0x8000), u16(0))))
}

func TestChangeActivePointerGrabEncode(t *testing.T) {
	checkEncode(t, &ChangeActivePointerGrabRequest{Cursor: 0x30, Time: 0x123, EventMask: 0xFF},
		req(30, 0, cat(u32(0x30), u32(0x123), u16(0xFF), u16(0))))
}

func TestGrabKeyboardEncode(t *testing.T) {
	checkEncode(t, &GrabKeyboardRequest{OwnerEvents: true, GrabWindow: 0x10, Time: 0x123,
		PointerMode: GrabModeAsync, KeyboardMode: GrabModeSync},
		req(31, 1, cat(u32(0x10), u32(0x123), u8(1), u8(0), u16(0))))
}

func TestGrabKeyboardReply(t *testing.T) {
	got := &GrabKeyboardReply{}
	if err := got.Parse(makeReply(2, 0, make([]byte, 24))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GrabKeyboardReply{Status: 2})
}

func TestUngrabKeyboardEncode(t *testing.T) {
	checkEncode(t, &UngrabKeyboardRequest{Time: 0x123}, req(32, 0, u32(0x123)))
}

func TestGrabKeyEncode(t *testing.T) {
	checkEncode(t, &GrabKeyRequest{OwnerEvents: true, GrabWindow: 0x10, Modifiers: 0x8000, Key: 24,
		PointerMode: GrabModeAsync, KeyboardMode: GrabModeSync},
		req(33, 1, cat(u32(0x10), u16(0x8000), u8(24), u8(1), u8(0), u8(0), u16(0))))
}

func TestUngrabKeyEncode(t *testing.T) {
	checkEncode(t, &UngrabKeyRequest{Key: 24, GrabWindow: 0x10, Modifiers: 0x8000},
		req(34, 24, cat(u32(0x10), u16(0x8000), u16(0))))
}

func TestAllowEventsEncode(t *testing.T) {
	checkEncode(t, &AllowEventsRequest{Mode: AllowReplayPointer, Time: 0x123}, req(35, 2, u32(0x123)))
}

func TestGrabServerEncode(t *testing.T) {
	checkEncode(t, &GrabServerRequest{}, req(36, 0, nil))
}

func TestUngrabServerEncode(t *testing.T) {
	checkEncode(t, &UngrabServerRequest{}, req(37, 0, nil))
}
