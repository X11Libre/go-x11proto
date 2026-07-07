package term

import "testing"

func TestEncodeCellLatin1PassesThroughAsRawByte(t *testing.T) {
	// Latin-1 core fonts index glyphs by the code point itself, so a rune in
	// 0-0xFF must become that exact byte value, NOT its UTF-8 encoding (which
	// for anything above 0x7F is 2+ bytes and would corrupt the line — see
	// TestCellsTextDoesNotUTF8Encode).
	if got := encodeCell('A'); got != 'A' {
		t.Errorf("encodeCell('A') = %v, want 'A'", got)
	}
	if got := encodeCell('é'); got != 0xE9 {
		t.Errorf("encodeCell('é') = %#x, want 0xE9 (Latin-1 code point, not UTF-8)", got)
	}
}

func TestEncodeCellZeroIsSpace(t *testing.T) {
	if got := encodeCell(0); got != ' ' {
		t.Errorf("encodeCell(0) = %q, want space", got)
	}
}

func TestEncodeCellBoxDrawingFallsBackToASCII(t *testing.T) {
	// All of these are above 0xFF (Latin-1), so they have no glyph in a core
	// bitmap font and must degrade to a look-alike. '°'/'±' are deliberately
	// NOT in this set: both are Latin-1 code points (U+00B0/U+00B1 alias
	// ISO-8859-1 0xB0/0xB1 exactly), so a Latin-1 font already has real
	// glyphs for them — encodeCell must pass those through as-is, not
	// substitute 'o'/'~'.
	cases := map[rune]byte{
		'─': '-', '│': '|', '┌': '+', '┐': '+', '└': '+', '┘': '+',
		'▒': '#', '≤': '<', '≥': '>', 'π': 'p',
		// double-line variants (mc's active-panel border, ncurses ACS_*
		// often emitted as real UTF-8 in a UTF-8 locale rather than via the
		// VT100 DEC Special Graphics escape mechanism)
		'═': '-', '║': '|', '╔': '+', '╗': '+', '╚': '+', '╝': '+',
	}
	for r, want := range cases {
		if got := encodeCell(r); got != want {
			t.Errorf("encodeCell(%q) = %q, want %q", r, got, want)
		}
	}
}

func TestEncodeCellLatin1SupplementSymbolsPassThrough(t *testing.T) {
	// '°' (U+00B0) and '±' (U+00B1) coincide with real ISO-8859-1 code
	// points, so a Latin-1 "fixed" font can render them directly.
	if got := encodeCell('°'); got != 0xB0 {
		t.Errorf("encodeCell('°') = %#x, want 0xB0", got)
	}
	if got := encodeCell('±'); got != 0xB1 {
		t.Errorf("encodeCell('±') = %#x, want 0xB1", got)
	}
}

func TestEncodeCellUnknownHighRuneIsQuestionMark(t *testing.T) {
	if got := encodeCell('日'); got != '?' {
		t.Errorf("encodeCell('日') = %q, want '?'", got)
	}
}

func TestCellsTextDoesNotUTF8Encode(t *testing.T) {
	cells := []Cell{{Rune: 'a'}, {Rune: '─'}, {Rune: 'b'}}
	got := cellsText(cells)
	// Must be exactly 3 bytes (one per cell) — a naive string(runes) would
	// produce 5 bytes here since '─' UTF-8-encodes to 3 bytes on its own,
	// which is exactly the bug this fixes: it would shift every following
	// cell's glyph left by 2 columns' worth of bytes.
	if len(got) != 3 {
		t.Fatalf("cellsText length = %d, want 3 (one byte per cell)", len(got))
	}
	if got != "a-b" {
		t.Fatalf("cellsText = %q, want %q", got, "a-b")
	}
}

// scrolledTerm builds a bare Term (no window, no font, no connection —
// ScrollTo's internal Draw() call no-ops when both AAFace and gc are nil, so
// this is safe to exercise offline) with rows lines of live content ending
// in "…D","E" and enough scrollback behind it (oldest first) to test
// visibleRows' windowing against a known, fully-controlled history.
func scrolledTerm(rows int) *Term {
	g := NewGrid(rows, 1)
	for _, r := range []rune{'A', 'B', 'C', 'D', 'E'} {
		g.cur[rows-1][0] = Cell{Rune: r}
		g.ScrollUp(1)
	}
	return &Term{grid: g, rows: rows}
}

func TestVisibleRowsLiveWhenNotScrolled(t *testing.T) {
	term := scrolledTerm(2)
	live := term.grid.cur
	got := term.visibleRows(live)
	if len(got) != 2 || text(got[0]) != "E" || text(got[1]) != " " {
		t.Fatalf("visibleRows(offset 0) = %q,%q, want live unchanged (\"E\",\" \")", text(got[0]), text(got[1]))
	}
}

func TestVisibleRowsBlendsScrollbackAndLive(t *testing.T) {
	// scrolledTerm(2) pushed blank,A,B,C,D into scrollback (oldest first)
	// and left live = ["E", blank].
	term := scrolledTerm(2)
	live := term.grid.cur

	term.scrollOffset = 1 // 1 line back: reveal 'D' at top, drop live's blank bottom row
	got := term.visibleRows(live)
	if text(got[0]) != "D" || text(got[1]) != "E" {
		t.Errorf("offset 1 = %q,%q, want \"D\",\"E\"", text(got[0]), text(got[1]))
	}

	term.scrollOffset = 2 // fully within scrollback: "C","D"
	got = term.visibleRows(live)
	if text(got[0]) != "C" || text(got[1]) != "D" {
		t.Errorf("offset 2 = %q,%q, want \"C\",\"D\"", text(got[0]), text(got[1]))
	}

	term.scrollOffset = 3 // offset > rows: still entirely within scrollback, one further back
	got = term.visibleRows(live)
	if text(got[0]) != "B" || text(got[1]) != "C" {
		t.Errorf("offset 3 = %q,%q, want \"B\",\"C\"", text(got[0]), text(got[1]))
	}
}

func TestScrollToClampsToAvailableHistory(t *testing.T) {
	term := scrolledTerm(2)
	max := term.grid.ScrollbackLen()

	term.ScrollTo(-5)
	if term.scrollOffset != 0 {
		t.Errorf("ScrollTo(-5): offset = %d, want 0 (clamped)", term.scrollOffset)
	}
	term.ScrollTo(max + 100)
	if term.scrollOffset != max {
		t.Errorf("ScrollTo(past end): offset = %d, want %d (clamped to available history)", term.scrollOffset, max)
	}
	term.ScrollBy(-100)
	if term.scrollOffset != 0 {
		t.Errorf("ScrollBy(-100) from max: offset = %d, want 0 (clamped)", term.scrollOffset)
	}
}
