package request

import "testing"

func TestCreateCursorEncode(t *testing.T) {
	checkEncode(t, &CreateCursorRequest{Cid: 0x10, Source: 0x20, Mask: 0x30,
		ForeRed: 1, ForeGreen: 2, ForeBlue: 3, BackRed: 4, BackGreen: 5, BackBlue: 6, X: 7, Y: 8},
		req(93, 0, cat(u32(0x10), u32(0x20), u32(0x30), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6), u16(7), u16(8))))
}

func TestCreateGlyphCursorEncode(t *testing.T) {
	checkEncode(t, &CreateGlyphCursorRequest{Cid: 0x10, SourceFont: 0x20, MaskFont: 0x30,
		SourceChar: 65, MaskChar: 66, ForeRed: 1, ForeGreen: 2, ForeBlue: 3, BackRed: 4, BackGreen: 5, BackBlue: 6},
		req(94, 0, cat(u32(0x10), u32(0x20), u32(0x30), u16(65), u16(66), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6))))
}

func TestFreeCursorEncode(t *testing.T) {
	checkEncode(t, &FreeCursorRequest{Cursor: 0x10}, req(95, 0, u32(0x10)))
}

func TestRecolorCursorEncode(t *testing.T) {
	checkEncode(t, &RecolorCursorRequest{Cursor: 0x10, ForeRed: 1, ForeGreen: 2, ForeBlue: 3, BackRed: 4, BackGreen: 5, BackBlue: 6},
		req(96, 0, cat(u32(0x10), u16(1), u16(2), u16(3), u16(4), u16(5), u16(6))))
}
