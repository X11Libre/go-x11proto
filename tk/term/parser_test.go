package term

import "testing"

func newParser(rows, cols int, typ Type) (*Grid, *Parser) {
	g := NewGrid(rows, cols)
	return g, NewParser(g, typ)
}

func TestFeedPlainText(t *testing.T) {
	g, p := newParser(2, 10, XTerm256Color)
	p.Feed([]byte("hello"))
	if text(g.cur[0])[:5] != "hello" {
		t.Fatalf("row0 = %q, want prefix %q", text(g.cur[0]), "hello")
	}
	if g.CursorCol != 5 {
		t.Errorf("CursorCol = %d, want 5", g.CursorCol)
	}
}

func TestFeedCRLF(t *testing.T) {
	g, p := newParser(2, 10, XTerm256Color)
	p.Feed([]byte("ab\r\ncd"))
	if text(g.cur[0])[:2] != "ab" || text(g.cur[1])[:2] != "cd" {
		t.Fatalf("rows = %q / %q, want ab / cd", text(g.cur[0]), text(g.cur[1]))
	}
}

func TestFeedBackspace(t *testing.T) {
	g, p := newParser(1, 10, XTerm256Color)
	p.Feed([]byte("ab\bc"))
	if text(g.cur[0])[:2] != "ac" {
		t.Fatalf("row0 = %q, want prefix %q", text(g.cur[0]), "ac")
	}
}

func TestFeedTab(t *testing.T) {
	g, p := newParser(1, 20, XTerm256Color)
	p.Feed([]byte("a\tb"))
	if g.cur[0][8].Rune != 'b' {
		t.Errorf("tab should land 'b' at column 8, got row %q", text(g.cur[0]))
	}
}

func TestFeedAutoWrap(t *testing.T) {
	g, p := newParser(2, 3, XTerm256Color)
	p.Feed([]byte("abcd"))
	if text(g.cur[0]) != "abc" || g.cur[1][0].Rune != 'd' {
		t.Fatalf("rows = %q / %q, want wrap of 'd' onto row 1", text(g.cur[0]), text(g.cur[1]))
	}
}

func TestFeedAutoWrapOffOverwritesLastColumn(t *testing.T) {
	g, p := newParser(2, 3, XTerm256Color)
	p.Feed([]byte("\x1b[?7l")) // DECRST 7: autowrap off
	p.Feed([]byte("abcd"))
	if text(g.cur[0]) != "abd" {
		t.Fatalf("row0 = %q, want %q ('d' overwriting the clamped last column, no wrap)", text(g.cur[0]), "abd")
	}
	if g.cur[1][0].Rune != 0 {
		t.Errorf("row1 should be untouched with autowrap off, got %q", text(g.cur[1]))
	}
}

func TestFeedCursorPosition(t *testing.T) {
	g, p := newParser(5, 5, XTerm256Color)
	p.Feed([]byte("\x1b[3;2H"))
	if g.CursorRow != 2 || g.CursorCol != 1 {
		t.Fatalf("cursor = (%d,%d), want (2,1) (1-based CUP -> 0-based)", g.CursorRow, g.CursorCol)
	}
}

func TestFeedCursorMoves(t *testing.T) {
	g, p := newParser(5, 5, XTerm256Color)
	p.Feed([]byte("\x1b[3;3H\x1b[1A\x1b[2C\x1b[1B\x1b[1D"))
	// start (2,2) 0-based; up 1 -> (1,2); right 2 -> (1,4); down 1 -> (2,4); left 1 -> (2,3)
	if g.CursorRow != 2 || g.CursorCol != 3 {
		t.Fatalf("cursor = (%d,%d), want (2,3)", g.CursorRow, g.CursorCol)
	}
}

func TestFeedEraseDisplay(t *testing.T) {
	g, p := newParser(2, 2, XTerm256Color)
	p.Feed([]byte("abcd\x1b[H\x1b[2J"))
	for r := range g.cur {
		if text(g.cur[r]) != "  " {
			t.Errorf("row %d = %q, want blank after ED 2", r, text(g.cur[r]))
		}
	}
}

// TestEraseFillsWithCurrentSGRBackground is a regression test for a real
// mc rendering bug found 2026-07-07: mc paints its top menu bar by setting a
// background colour via SGR and erasing to the end of the line to fill the
// rest of the row in it (the standard way full-width status/menu bars are
// drawn) — Grid.blank's doc comment already promised erasing with "the
// active SGR background, not always black", but nothing kept
// Grid.DefaultFg/DefaultBg (what blank() actually reads) in sync with the
// parser's current SGR colours, so the erased portion silently fell back to
// the terminal's startup default (black) instead — a black gap after the
// last character on an otherwise fully-coloured bar.
func TestEraseFillsWithCurrentSGRBackground(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	p.Feed([]byte("\x1b[44mab\x1b[K")) // blue bg, "ab", erase to end of line
	want := Color{Mode: ColorIndexed, Index: 4}
	for c := 0; c < 5; c++ {
		if g.cur[0][c].Bg != want {
			t.Errorf("cell %d Bg = %+v, want %+v (active SGR background)", c, g.cur[0][c].Bg, want)
		}
	}
}

func TestSGRBasicColorAndReset(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	p.Feed([]byte("\x1b[31;1mx\x1b[0my"))
	if g.cur[0][0].Fg != (Color{Mode: ColorIndexed, Index: 1}) {
		t.Errorf("fg = %+v, want red index 1", g.cur[0][0].Fg)
	}
	if g.cur[0][0].Attr&AttrBold == 0 {
		t.Errorf("expected bold attribute on first cell")
	}
	if g.cur[0][1].Fg != (Color{}) || g.cur[0][1].Attr != 0 {
		t.Errorf("SGR 0 should reset style for the second cell, got fg=%+v attr=%v", g.cur[0][1].Fg, g.cur[0][1].Attr)
	}
}

func TestSGR256Color(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	p.Feed([]byte("\x1b[38;5;200mx"))
	if g.cur[0][0].Fg != (Color{Mode: ColorIndexed, Index: 200}) {
		t.Errorf("fg = %+v, want indexed 200", g.cur[0][0].Fg)
	}
}

func TestSGRTruecolor(t *testing.T) {
	g, p := newParser(1, 5, XTermTrueColor)
	p.Feed([]byte("\x1b[38;2;10;20;30mx"))
	want := Color{Mode: ColorRGB, R: 10, G: 20, B: 30}
	if g.cur[0][0].Fg != want {
		t.Errorf("fg = %+v, want %+v", g.cur[0][0].Fg, want)
	}
}

func TestSGRTruecolorClampedOn256ColorType(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	p.Feed([]byte("\x1b[38;2;255;0;0mx")) // pure red
	fg := g.cur[0][0].Fg
	if fg.Mode != ColorIndexed {
		t.Fatalf("truecolor request on a 256-colour Type should be quantized to indexed, got %+v", fg)
	}
}

func TestSGRColorClampedAwayOnVT100(t *testing.T) {
	g, p := newParser(1, 5, VT100)
	p.Feed([]byte("\x1b[31mx"))
	if g.cur[0][0].Fg != (Color{}) {
		t.Errorf("VT100 has Colors==0, SGR colour should be dropped entirely, got %+v", g.cur[0][0].Fg)
	}
}

func TestPrivateModeCursorVisibility(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	p.Feed([]byte("\x1b[?25l"))
	if g.CursorVisible {
		t.Errorf("CSI ?25l should hide the cursor")
	}
	p.Feed([]byte("\x1b[?25h"))
	if !g.CursorVisible {
		t.Errorf("CSI ?25h should show the cursor")
	}
}

func TestAltScreenGatedByType(t *testing.T) {
	g, p := newParser(2, 2, XTerm256Color)
	p.Feed([]byte("\x1b[?1049h"))
	if !g.OnAltScreen() {
		t.Fatalf("XTerm256Color supports AltScreen, ?1049h should switch")
	}
	p.Feed([]byte("\x1b[?1049l"))
	if g.OnAltScreen() {
		t.Fatalf("?1049l should restore the primary screen")
	}

	g2, p2 := newParser(2, 2, VT100)
	p2.Feed([]byte("\x1b[?1049h"))
	if g2.OnAltScreen() {
		t.Errorf("VT100 does not support AltScreen; ?1049h must be ignored")
	}
}

func TestAppCursorModeGatedByType(t *testing.T) {
	_, p := newParser(2, 2, XTerm256Color)
	p.Feed([]byte("\x1b[?1h"))
	if !p.Modes.AppCursor {
		t.Errorf("XTerm256Color supports AppCursor, ?1h should set it")
	}

	_, p2 := newParser(2, 2, VT100)
	p2.Feed([]byte("\x1b[?1h"))
	if p2.Modes.AppCursor {
		t.Errorf("VT100 does not advertise AppCursor; ?1h must be ignored")
	}
}

func TestBracketedPasteMode(t *testing.T) {
	_, p := newParser(2, 2, XTerm256Color)
	p.Feed([]byte("\x1b[?2004h"))
	if !p.Modes.BracketedPaste {
		t.Errorf("?2004h should enable bracketed paste")
	}
	p.Feed([]byte("\x1b[?2004l"))
	if p.Modes.BracketedPaste {
		t.Errorf("?2004l should disable bracketed paste")
	}
}

func TestDSRCursorPositionReport(t *testing.T) {
	g, p := newParser(5, 5, XTerm256Color)
	var got []byte
	p.Respond = func(b []byte) { got = append(got, b...) }
	p.Feed([]byte("\x1b[3;4H\x1b[6n"))
	want := "\x1b[3;4R" // 0-based (2,3) reported back 1-based
	if string(got) != want {
		t.Errorf("DSR reply = %q, want %q", got, want)
	}
	_ = g
}

func TestDeviceAttributesResponds(t *testing.T) {
	_, p := newParser(2, 2, XTerm256Color)
	var got []byte
	p.Respond = func(b []byte) { got = append(got, b...) }
	p.Feed([]byte("\x1b[c"))
	if len(got) == 0 {
		t.Errorf("CSI c (DA) should produce a response")
	}
}

func TestOSCSetTitle(t *testing.T) {
	var title string
	_, p := newParser(2, 2, XTerm256Color)
	p.SetTitle = func(s string) { title = s }
	p.Feed([]byte("\x1b]0;my title\x07"))
	if title != "my title" {
		t.Errorf("title = %q, want %q", title, "my title")
	}
}

func TestOSCSetTitleTerminatedByST(t *testing.T) {
	var title string
	_, p := newParser(2, 2, XTerm256Color)
	p.SetTitle = func(s string) { title = s }
	p.Feed([]byte("\x1b]2;other title\x1b\\"))
	if title != "other title" {
		t.Errorf("title = %q, want %q", title, "other title")
	}
}

func TestUTF8SplitAcrossFeedCalls(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	euro := "€" // 3-byte UTF-8
	b := []byte(euro)
	p.Feed(b[:1])
	p.Feed(b[1:])
	if g.cur[0][0].Rune != '€' {
		t.Errorf("cell = %q, want euro sign (split across two Feed calls)", string(g.cur[0][0].Rune))
	}
}

// TestCharsetDesignationDoesNotEatFollowingText is a regression test: an
// earlier version of the state machine mishandled ESC ( / ) / * / + (G0-G3
// charset designation) by reusing the "waiting for ST" flag, which meant the
// designator's following byte (almost always 'B' for US-ASCII, e.g. the
// \e(B many shells/terminfo setups emit) flipped the parser into the
// DCS/OSC-ignore state instead of returning to ground — silently eating
// every subsequent character as if it were inside an unterminated string.
func TestCharsetDesignationDoesNotEatFollowingText(t *testing.T) {
	g, p := newParser(1, 10, XTerm256Color)
	p.Feed([]byte("\x1b(Bhello"))
	if text(g.cur[0])[:5] != "hello" {
		t.Fatalf("row0 = %q, want %q — charset designator must not swallow following text", text(g.cur[0]), "hello")
	}
}

func TestUnknownCSIDoesNotCrashAndResyncs(t *testing.T) {
	g, p := newParser(1, 10, XTerm256Color)
	p.Feed([]byte("\x1b[9;9;9zhello")) // 'z' is not a final byte this Parser handles
	if text(g.cur[0])[:5] != "hello" {
		t.Fatalf("row0 = %q, want %q — an unrecognised CSI must still resync to ground", text(g.cur[0]), "hello")
	}
}

func TestInsertAndDeleteChars(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	p.Feed([]byte("abcde\x1b[3D\x1b[2@")) // cursor back to col 2, insert 2 blanks
	if text(g.cur[0]) != "ab  c" {
		t.Fatalf("row0 = %q, want %q", text(g.cur[0]), "ab  c")
	}
	p.Feed([]byte("\x1b[2P")) // delete the 2 blanks just inserted
	if text(g.cur[0]) != "abc  " {
		t.Fatalf("row0 after DCH = %q, want %q", text(g.cur[0]), "abc  ")
	}
}

func TestInsertAndDeleteLines(t *testing.T) {
	g, p := newParser(3, 2, XTerm256Color)
	p.Feed([]byte("aa\r\nbb\r\ncc\x1b[2;1H\x1b[1L")) // cursor to row1(0-based), insert 1 line
	if text(g.cur[1]) != "  " || text(g.cur[2]) != "bb" {
		t.Fatalf("after IL: rows = %q / %q / %q", text(g.cur[0]), text(g.cur[1]), text(g.cur[2]))
	}
}

func TestScrollRegionConstrainsNewline(t *testing.T) {
	g, p := newParser(4, 6, XTerm256Color)
	p.Feed([]byte("\x1b[2;3r"))       // region rows 2-3 (1-based) -> 0-based [1,2]
	p.Feed([]byte("\x1b[4;1Hbottom")) // write on row 4 (outside region, exactly fills the 6 cols)
	if text(g.cur[3]) != "bottom" {
		t.Fatalf("row3 = %q, want %q", text(g.cur[3]), "bottom")
	}
}

// TestDECSpecialGraphicsBoxDrawing is a regression test for the mc/ncurses
// rendering bug found 2026-07-06: apps that draw box borders via the classic
// smacs=\E(0/rmacs=\E(B terminfo mechanism (not raw UTF-8) need G0 tracked
// so the designated bytes decode to the right box-drawing runes instead of
// their literal ASCII letters.
func TestDECSpecialGraphicsBoxDrawing(t *testing.T) {
	g, p := newParser(1, 10, XTerm256Color)
	p.Feed([]byte("\x1b(0lqqqk\x1b(Bhi"))
	got := text(g.cur[0])
	want := "┌───┐" + "hi" + "   " // lqqqk -> ┌───┐, back to ASCII for "hi"
	if got != want {
		t.Fatalf("row0 = %q, want %q", got, want)
	}
}

func TestDECSpecialGraphicsOnlyAppliesToG0(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	// Designating G1 (')') as special graphics must not affect G0/plain text.
	p.Feed([]byte("\x1b)0abc"))
	if text(g.cur[0])[:3] != "abc" {
		t.Fatalf("row0 = %q, want plain %q (G1 designation must not alter G0 output)", text(g.cur[0]), "abc")
	}
}

func TestResetClearsAltCharset(t *testing.T) {
	g, p := newParser(1, 5, XTerm256Color)
	p.Feed([]byte("\x1b(0")) // enable DEC special graphics
	p.Feed([]byte("\x1bc"))  // RIS: full reset
	p.Feed([]byte("q"))
	if g.cur[0][0].Rune != 'q' {
		t.Fatalf("after RIS, 'q' should print literally, got %q", string(g.cur[0][0].Rune))
	}
}
