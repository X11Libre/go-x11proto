package term

import "testing"

func text(row []Cell) string {
	s := make([]rune, len(row))
	for i, c := range row {
		if c.Rune == 0 {
			s[i] = ' '
		} else {
			s[i] = c.Rune
		}
	}
	return string(s)
}

func TestNewGridBlank(t *testing.T) {
	g := NewGrid(3, 5)
	if g.Rows != 3 || g.Cols != 5 {
		t.Fatalf("dims = %dx%d, want 3x5", g.Rows, g.Cols)
	}
	if g.Cell(0, 0).Rune != 0 {
		t.Errorf("fresh grid should be all-zero cells")
	}
	if !g.CursorVisible {
		t.Errorf("cursor should start visible")
	}
}

func TestPutRuneAdvances(t *testing.T) {
	g := NewGrid(2, 3)
	g.PutRune('a', Color{}, Color{}, 0, "")
	g.PutRune('b', Color{}, Color{}, 0, "")
	g.PutRune('c', Color{}, Color{}, 0, "")
	if g.CursorCol != 3 {
		t.Fatalf("CursorCol = %d, want 3 (past edge; Grid itself never wraps)", g.CursorCol)
	}
	if text(g.cur[0]) != "abc" {
		t.Errorf("row0 = %q, want %q", text(g.cur[0]), "abc")
	}
}

func TestPutRuneClampsInsteadOfWrapping(t *testing.T) {
	// Grid.PutRune has no notion of AutoWrap: past the last column it clamps
	// back onto it, so repeated writes overwrite the same cell — wrapping
	// (or not) is entirely the Parser's decision. See TestFeedAutoWrap and
	// TestFeedAutoWrapOffOverwritesLastColumn in parser_test.go for the
	// behaviour actually driven through Feed.
	g := NewGrid(1, 2)
	g.PutRune('a', Color{}, Color{}, 0, "")
	g.PutRune('b', Color{}, Color{}, 0, "")
	g.PutRune('c', Color{}, Color{}, 0, "")
	if text(g.cur[0]) != "ac" {
		t.Errorf("row0 = %q, want %q ('c' overwriting the clamped last column)", text(g.cur[0]), "ac")
	}
}

func TestScrollUpPushesScrollbackAndBlanksBottom(t *testing.T) {
	g := NewGrid(2, 3)
	g.cur[0][0] = Cell{Rune: 'x'}
	g.cur[1][0] = Cell{Rune: 'y'}
	g.ScrollUp(1)
	if text(g.cur[0]) != "y  " {
		t.Errorf("row0 after scroll = %q, want %q", text(g.cur[0]), "y  ")
	}
	if text(g.cur[1]) != "   " {
		t.Errorf("row1 after scroll should be blank, got %q", text(g.cur[1]))
	}
	sb := g.ScrollbackLines(1)
	if len(sb) != 1 || sb[0][0].Rune != 'x' {
		t.Errorf("scrollback should hold the scrolled-off row 'x...', got %+v", sb)
	}
}

func TestScrollUpRestrictedToRegionNoScrollback(t *testing.T) {
	g := NewGrid(4, 3)
	g.SetScrollRegion(1, 2)
	g.cur[0][0] = Cell{Rune: 'a'}
	g.cur[1][0] = Cell{Rune: 'b'}
	g.cur[2][0] = Cell{Rune: 'c'}
	g.cur[3][0] = Cell{Rune: 'd'}
	g.ScrollUp(1)
	if g.cur[0][0].Rune != 'a' || g.cur[3][0].Rune != 'd' {
		t.Errorf("rows outside the scroll region must be untouched")
	}
	if g.cur[1][0].Rune != 'c' {
		t.Errorf("row1 should now hold former row2 ('c'), got %q", string(g.cur[1][0].Rune))
	}
	if len(g.scrollback) != 0 {
		t.Errorf("a partial-region scroll must not push scrollback, got %d lines", len(g.scrollback))
	}
}

func TestScrollDownFillsTopNoScrollback(t *testing.T) {
	g := NewGrid(2, 3)
	g.cur[0][0] = Cell{Rune: 'x'}
	g.cur[1][0] = Cell{Rune: 'y'}
	g.ScrollDown(1)
	if g.cur[1][0].Rune != 'x' {
		t.Errorf("row1 should now hold former row0, got %q", string(g.cur[1][0].Rune))
	}
	if g.cur[0][0].Rune != ' ' {
		t.Errorf("row0 should be blanked, got %q", string(g.cur[0][0].Rune))
	}
	if len(g.scrollback) != 0 {
		t.Errorf("ScrollDown must never touch scrollback")
	}
}

func TestEraseLineModes(t *testing.T) {
	g := NewGrid(1, 5)
	for i := range g.cur[0] {
		g.cur[0][i] = Cell{Rune: 'x'}
	}
	g.CursorCol = 2
	g.EraseLine(EraseToEnd)
	if text(g.cur[0]) != "xx   " {
		t.Errorf("EraseToEnd = %q, want %q", text(g.cur[0]), "xx   ")
	}

	g2 := NewGrid(1, 5)
	for i := range g2.cur[0] {
		g2.cur[0][i] = Cell{Rune: 'x'}
	}
	g2.CursorCol = 2
	g2.EraseLine(EraseToStart)
	if text(g2.cur[0]) != "   xx" {
		t.Errorf("EraseToStart = %q, want %q", text(g2.cur[0]), "   xx")
	}
}

func TestEraseDisplayAll(t *testing.T) {
	g := NewGrid(2, 2)
	for r := range g.cur {
		for c := range g.cur[r] {
			g.cur[r][c] = Cell{Rune: 'x'}
		}
	}
	g.EraseDisplay(EraseAll)
	for r := range g.cur {
		if text(g.cur[r]) != "  " {
			t.Errorf("row %d = %q, want blank", r, text(g.cur[r]))
		}
	}
}

func TestSetCursorClamps(t *testing.T) {
	g := NewGrid(3, 3)
	g.SetCursor(100, -5)
	if g.CursorRow != 2 || g.CursorCol != 0 {
		t.Errorf("cursor = (%d,%d), want clamped to (2,0)", g.CursorRow, g.CursorCol)
	}
}

func TestSaveRestoreCursor(t *testing.T) {
	g := NewGrid(5, 5)
	g.SetCursor(2, 3)
	g.SaveCursor()
	g.SetCursor(0, 0)
	g.RestoreCursor()
	if g.CursorRow != 2 || g.CursorCol != 3 {
		t.Errorf("cursor after restore = (%d,%d), want (2,3)", g.CursorRow, g.CursorCol)
	}
}

func TestAltScreenSwapAndRestore(t *testing.T) {
	g := NewGrid(2, 2)
	g.cur[0][0] = Cell{Rune: 'p'} // primary content
	g.EnterAltScreen()
	if !g.OnAltScreen() {
		t.Fatalf("should be on alt screen")
	}
	if g.cur[0][0].Rune != ' ' {
		t.Errorf("alt screen should start blank, got %q", string(g.cur[0][0].Rune))
	}
	g.cur[0][0] = Cell{Rune: 'a'} // alt-screen content
	g.ExitAltScreen()
	if g.OnAltScreen() {
		t.Fatalf("should be back on primary")
	}
	if g.cur[0][0].Rune != 'p' {
		t.Errorf("primary content should be preserved under the alt screen, got %q", string(g.cur[0][0].Rune))
	}
}

func TestResizeTruncatesAndPads(t *testing.T) {
	g := NewGrid(2, 2)
	g.cur[0][0] = Cell{Rune: 'a'}
	g.cur[1][1] = Cell{Rune: 'b'}
	g.Resize(3, 3)
	if g.Rows != 3 || g.Cols != 3 {
		t.Fatalf("dims after grow = %dx%d, want 3x3", g.Rows, g.Cols)
	}
	if g.cur[0][0].Rune != 'a' || g.cur[1][1].Rune != 'b' {
		t.Errorf("existing content should survive a grow")
	}
	g.Resize(1, 1)
	if g.cur[0][0].Rune != 'a' {
		t.Errorf("surviving corner cell should still be 'a' after shrink")
	}
}

func TestSetScrollRegionInvalidResetsToWhole(t *testing.T) {
	g := NewGrid(5, 5)
	g.SetScrollRegion(3, 1) // bot < top: invalid
	if g.scrollTop != 0 || g.scrollBot != 4 {
		t.Errorf("invalid region should reset to whole screen, got [%d,%d]", g.scrollTop, g.scrollBot)
	}
}
