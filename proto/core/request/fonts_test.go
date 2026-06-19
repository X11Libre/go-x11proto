package request

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
)

// ci builds a CHARINFO on the wire (6 x INT16/CARD16).
func ci(a, b, c, d, e, f uint16) []byte {
	return cat(u16(a), u16(b), u16(c), u16(d), u16(e), u16(f))
}

func TestCloseFontEncode(t *testing.T) {
	checkEncode(t, &CloseFontRequest{Font: 0x10}, req(46, 0, u32(0x10)))
}

func TestQueryFontEncode(t *testing.T) {
	checkEncode(t, &QueryFontRequest{Font: 0x10}, req(47, 0, u32(0x10)))
}

func TestQueryFontReply(t *testing.T) {
	tail := cat(
		ci(1, 2, 3, 4, 5, 6), make([]byte, 4),
		ci(7, 8, 9, 10, 11, 12), make([]byte, 4),
		u16(32), u16(126), u16(65), u16(1),
		u8(0), u8(0), u8(0), u8(1),
		u16(10), u16(12), u32(1),
		u32(0xAA), u32(0xBB), // 1 property
		ci(13, 14, 15, 16, 17, 18), // 1 charinfo
	)
	got := &QueryFontReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryFontReply{
		MinBounds: base.CharInfo{LeftSideBearing: 1, RightSideBearing: 2, CharacterWidth: 3, Ascent: 4, Descent: 5, Attributes: 6}, MaxBounds: base.CharInfo{LeftSideBearing: 7, RightSideBearing: 8, CharacterWidth: 9, Ascent: 10, Descent: 11, Attributes: 12},
		MinCharOrByte2: 32, MaxCharOrByte2: 126, DefaultChar: 65,
		DrawDirection: 0, MinByte1: 0, MaxByte1: 0, AllCharsExist: true,
		FontAscent: 10, FontDescent: 12,
		Properties: []base.FontProp{{Name: 0xAA, Value: 0xBB}},
		CharInfos:  []base.CharInfo{{LeftSideBearing: 13, RightSideBearing: 14, CharacterWidth: 15, Ascent: 16, Descent: 17, Attributes: 18}},
	})
}

func TestQueryTextExtentsEncodeOdd(t *testing.T) {
	checkEncode(t, &QueryTextExtentsRequest{Font: 0x10, Text: []base.CARD16{0x4869}},
		req(48, 1, cat(u32(0x10), []byte{0x48, 0x69})))
}

func TestQueryTextExtentsReply(t *testing.T) {
	tail := cat(u16(10), u16(12), u16(8), u16(3), u32(100), u32(0xFFFFFFF0), u32(50), make([]byte, 4))
	got := &QueryTextExtentsReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &QueryTextExtentsReply{
		DrawDirection: 0, FontAscent: 10, FontDescent: 12, OverallAscent: 8, OverallDescent: 3,
		OverallWidth: 100, OverallLeft: -16, OverallRight: 50,
	})
}

func TestListFontsEncode(t *testing.T) {
	checkEncode(t, &ListFontsRequest{MaxNames: 10, Pattern: "*"},
		req(49, 0, cat(u16(10), u16(1), []byte("*"))))
}

func TestListFontsReply(t *testing.T) {
	tail := cat(u16(2), make([]byte, 22), u8(3), []byte("abc"), u8(2), []byte("de"))
	got := &ListFontsReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &ListFontsReply{Names: []string{"abc", "de"}})
}

func TestGetFontPathEncode(t *testing.T) {
	checkEncode(t, &GetFontPathRequest{}, req(52, 0, nil))
}

func TestGetFontPathReply(t *testing.T) {
	tail := cat(u16(1), make([]byte, 22), u8(4), []byte("/foo"))
	got := &GetFontPathReply{}
	if err := got.Parse(makeReply(0, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &GetFontPathReply{Path: []string{"/foo"}})
}

func TestSetFontPathEncode(t *testing.T) {
	checkEncode(t, &SetFontPathRequest{Path: []string{"/a", "/bb"}},
		req(51, 0, cat(u16(2), u16(0), u8(2), []byte("/a"), u8(3), []byte("/bb"))))
}

func TestListFontsWithInfoEncode(t *testing.T) {
	checkEncode(t, &ListFontsWithInfoRequest{MaxNames: 5, Pattern: "*"},
		req(50, 0, cat(u16(5), u16(1), []byte("*"))))
}

func TestListFontsWithInfoReply(t *testing.T) {
	tail := cat(
		ci(1, 2, 3, 4, 5, 6), make([]byte, 4),
		ci(7, 8, 9, 10, 11, 12), make([]byte, 4),
		u16(32), u16(126), u16(65), u16(0), // 0 props
		u8(0), u8(0), u8(0), u8(1),
		u16(10), u16(12), u32(99),
		[]byte("Helv"),
	)
	got := &ListFontsWithInfoReply{}
	if err := got.Parse(makeReply(4, 0, tail)); err != nil {
		t.Fatal(err)
	}
	checkReply(t, got, &ListFontsWithInfoReply{
		LastReply: false,
		MinBounds: base.CharInfo{LeftSideBearing: 1, RightSideBearing: 2, CharacterWidth: 3, Ascent: 4, Descent: 5, Attributes: 6}, MaxBounds: base.CharInfo{LeftSideBearing: 7, RightSideBearing: 8, CharacterWidth: 9, Ascent: 10, Descent: 11, Attributes: 12},
		MinCharOrByte2: 32, MaxCharOrByte2: 126, DefaultChar: 65,
		DrawDirection: 0, MinByte1: 0, MaxByte1: 0, AllCharsExist: true,
		FontAscent: 10, FontDescent: 12, RepliesHint: 99,
		Properties: []base.FontProp{}, Name: "Helv",
	})
}
