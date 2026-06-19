package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

func TestQueryBestSizeEncode(t *testing.T) {
	checkEncode(t, &QueryBestSizeRequest{Class: BestSizeTile, Drawable: 0x10, Width: 100, Height: 50},
		req(97, 1, cat(u32(0x10), u16(100), u16(50))))
}

func TestQueryBestSizeReply(t *testing.T) {
	got := &QueryBestSizeReply{}
	if err := got.Parse(makeReply(0, 0, cat(u16(64), u16(64), make([]byte, 20)))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryBestSizeReply{Width: 64, Height: 64})
}

func TestChangeKeyboardMappingEncode(t *testing.T) {
	checkEncode(t, &ChangeKeyboardMappingRequest{FirstKeycode: 8, KeysymsPerKeycode: 2, KeycodeCount: 1,
		Keysyms: []base.CARD32{0x61, 0x41}},
		req(100, 1, cat(u8(8), u8(2), u16(0), u32(0x61), u32(0x41))))
}

func TestGetKeyboardMappingEncode(t *testing.T) {
	checkEncode(t, &GetKeyboardMappingRequest{FirstKeycode: 8, Count: 2}, req(101, 0, cat(u8(8), u8(2), u16(0))))
}

func TestGetKeyboardMappingReply(t *testing.T) {
	tail := cat(make([]byte, 24), u32(0x61), u32(0x41))
	got := &GetKeyboardMappingReply{}
	if err := got.Parse(makeReply(1, 2, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetKeyboardMappingReply{KeysymsPerKeycode: 1, Keysyms: []base.CARD32{0x61, 0x41}})
}

func TestChangeKeyboardControlEncode(t *testing.T) {
	checkEncode(t, &ChangeKeyboardControlRequest{
		ValueMask: KB_MASK_KEY_CLICK_PERCENT | KB_MASK_BELL_PERCENT, KeyClickPercent: 50, BellPercent: 80},
		req(102, 0, cat(u32(0x03), u32(50), u32(80))))
}

func TestGetKeyboardControlEncode(t *testing.T) {
	checkEncode(t, &GetKeyboardControlRequest{}, req(103, 0, nil))
}

func TestGetKeyboardControlReply(t *testing.T) {
	ar := make([]byte, 32)
	for i := range ar {
		ar[i] = byte(i)
	}
	tail := cat(u32(0xF), u8(50), u8(80), u16(400), u16(100), u16(0), ar)
	got := &GetKeyboardControlReply{}
	if err := got.Parse(makeReply(1, 5, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetKeyboardControlReply{GlobalAutoRepeat: true, LedMask: 0xF,
		KeyClickPercent: 50, BellPercent: 80, BellPitch: 400, BellDuration: 100, AutoRepeats: ar})
}

func TestBellEncode(t *testing.T) {
	checkEncode(t, &BellRequest{Percent: -50}, req(104, 0xCE, nil))
}

func TestChangePointerControlEncode(t *testing.T) {
	checkEncode(t, &ChangePointerControlRequest{AccelerationNumerator: 2, AccelerationDenominator: 1,
		Threshold: 4, DoAcceleration: true, DoThreshold: false},
		req(105, 0, cat(u16(2), u16(1), u16(4), u8(1), u8(0))))
}

func TestGetPointerControlEncode(t *testing.T) {
	checkEncode(t, &GetPointerControlRequest{}, req(106, 0, nil))
}

func TestGetPointerControlReply(t *testing.T) {
	got := &GetPointerControlReply{}
	if err := got.Parse(makeReply(0, 0, cat(u16(2), u16(1), u16(4), make([]byte, 18)))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetPointerControlReply{AccelerationNumerator: 2, AccelerationDenominator: 1, Threshold: 4})
}

func TestSetScreenSaverEncode(t *testing.T) {
	checkEncode(t, &SetScreenSaverRequest{Timeout: 600, Interval: 60, PreferBlanking: 1, AllowExposures: 1},
		req(107, 0, cat(u16(600), u16(60), u8(1), u8(1), u16(0))))
}

func TestGetScreenSaverEncode(t *testing.T) {
	checkEncode(t, &GetScreenSaverRequest{}, req(108, 0, nil))
}

func TestGetScreenSaverReply(t *testing.T) {
	got := &GetScreenSaverReply{}
	if err := got.Parse(makeReply(0, 0, cat(u16(600), u16(60), u8(1), u8(1), make([]byte, 18)))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetScreenSaverReply{Timeout: 600, Interval: 60, PreferBlanking: 1, AllowExposures: 1})
}

func TestChangeHostsEncode(t *testing.T) {
	checkEncode(t, &ChangeHostsRequest{Mode: HostInsert, Family: FamilyInternet, Address: []byte{127, 0, 0, 1}},
		req(109, 0, cat(u8(0), u8(0), u16(4), []byte{127, 0, 0, 1})))
}

func TestListHostsEncode(t *testing.T) {
	checkEncode(t, &ListHostsRequest{}, req(110, 0, nil))
}

func TestListHostsReply(t *testing.T) {
	tail := cat(u16(1), make([]byte, 22), u8(0), u8(0), u16(4), []byte{127, 0, 0, 1})
	got := &ListHostsReply{}
	if err := got.Parse(makeReply(1, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &ListHostsReply{Enabled: true, Hosts: []Host{{Family: 0, Address: []byte{127, 0, 0, 1}}}})
}

func TestSetAccessControlEncode(t *testing.T) {
	checkEncode(t, &SetAccessControlRequest{Mode: AccessControlEnable}, req(111, 1, nil))
}

func TestSetCloseDownModeEncode(t *testing.T) {
	checkEncode(t, &SetCloseDownModeRequest{Mode: CloseDownRetainPermanent}, req(112, 1, nil))
}

func TestKillClientEncode(t *testing.T) {
	checkEncode(t, &KillClientRequest{Resource: 0xAB}, req(113, 0, u32(0xAB)))
}

func TestRotatePropertiesEncode(t *testing.T) {
	checkEncode(t, &RotatePropertiesRequest{Window: 0x10, Delta: 1, Properties: []base.ATOM{0xA, 0xB}},
		req(114, 0, cat(u32(0x10), u16(2), u16(1), u32(0xA), u32(0xB))))
}

func TestForceScreenSaverEncode(t *testing.T) {
	checkEncode(t, &ForceScreenSaverRequest{Mode: ScreenSaverActivate}, req(115, 1, nil))
}

func TestSetPointerMappingEncode(t *testing.T) {
	checkEncode(t, &SetPointerMappingRequest{Map: []base.CARD8{1, 2, 3}}, req(116, 3, []byte{1, 2, 3}))
}

func TestSetPointerMappingReply(t *testing.T) {
	got := &SetPointerMappingReply{}
	if err := got.Parse(makeReply(0, 0, make([]byte, 24))); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &SetPointerMappingReply{Status: 0})
}

func TestGetPointerMappingEncode(t *testing.T) {
	checkEncode(t, &GetPointerMappingRequest{}, req(117, 0, nil))
}

func TestGetPointerMappingReply(t *testing.T) {
	tail := cat(make([]byte, 24), []byte{1, 2, 3})
	got := &GetPointerMappingReply{}
	if err := got.Parse(makeReply(3, 1, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetPointerMappingReply{Map: []byte{1, 2, 3}})
}

func TestSetModifierMappingEncode(t *testing.T) {
	kc := make([]base.CARD8, 16)
	for i := range kc {
		kc[i] = base.CARD8(i + 1)
	}
	want := make([]byte, 16)
	for i := range want {
		want[i] = byte(i + 1)
	}
	checkEncode(t, &SetModifierMappingRequest{KeycodesPerModifier: 2, Keycodes: kc}, req(118, 2, want))
}

func TestGetModifierMappingReply(t *testing.T) {
	body := make([]byte, 16)
	for i := range body {
		body[i] = byte(i + 1)
	}
	got := &GetModifierMappingReply{}
	if err := got.Parse(makeReply(2, 4, cat(make([]byte, 24), body))); err != nil {
		t.Fatal(err)
	}
	kc := make([]base.CARD8, 16)
	for i := range kc {
		kc[i] = base.CARD8(i + 1)
	}
	checkReply(t, got, &GetModifierMappingReply{KeycodesPerModifier: 2, Keycodes: kc})
}

func TestGetModifierMappingEncode(t *testing.T) {
	checkEncode(t, &GetModifierMappingRequest{}, req(119, 0, nil))
}

func TestNoOperationEncode(t *testing.T) {
	checkEncode(t, &NoOperationRequest{}, req(127, 0, nil))
}
