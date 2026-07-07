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
