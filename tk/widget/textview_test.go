package widget

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// newTV builds a TextView wired with a synthetic font (height only) and a
// buffer, with no server resources, so the editing logic can be driven offline.
func newTV(h int, lines ...string) *TextView {
	tv := &TextView{Font: &font.Font{Ascent: 10, Descent: 2}}
	tv.H = base.CARD16(h * tv.Font.Height())
	if len(lines) == 0 {
		lines = []string{""}
	}
	tv.lines = append([]string(nil), lines...)
	return tv
}

func typeRunes(tv *TextView, s string) {
	for _, r := range s {
		tv.edit(keyboard.Event{Rune: r})
	}
}

func key(tv *TextView, k keyboard.Key) { tv.edit(keyboard.Event{Key: k}) }

// seek positions the cursor and collapses the selection there (what a real
// click/navigation does), so tests start from a clean, unselected cursor.
func seek(tv *TextView, line, col int) {
	tv.curLine, tv.curCol = line, col
	tv.collapseSelection()
}

func TestTextViewInsertAndText(t *testing.T) {
	tv := newTV(5)
	typeRunes(tv, "hello")
	if got := tv.Text(); got != "hello" {
		t.Fatalf("Text = %q, want %q", got, "hello")
	}
	if tv.curCol != 5 {
		t.Errorf("curCol = %d, want 5", tv.curCol)
	}
}

func TestTextViewNewlineSplitsLine(t *testing.T) {
	tv := newTV(5, "abcdef")
	seek(tv, 0, 3)
	key(tv, keyboard.KeyEnter)
	if got := tv.Text(); got != "abc\ndef" {
		t.Fatalf("Text = %q, want %q", got, "abc\ndef")
	}
	if tv.curLine != 1 || tv.curCol != 0 {
		t.Errorf("cursor = (%d,%d), want (1,0)", tv.curLine, tv.curCol)
	}
}

func TestTextViewBackspaceJoinsLines(t *testing.T) {
	tv := newTV(5, "abc", "def")
	seek(tv, 1, 0)
	key(tv, keyboard.KeyBackspace)
	if got := tv.Text(); got != "abcdef" {
		t.Fatalf("Text = %q, want %q", got, "abcdef")
	}
	if tv.curLine != 0 || tv.curCol != 3 {
		t.Errorf("cursor = (%d,%d), want (0,3)", tv.curLine, tv.curCol)
	}
}

func TestTextViewBackspaceWithinLine(t *testing.T) {
	tv := newTV(5, "hello")
	seek(tv, 0, 5)
	key(tv, keyboard.KeyBackspace)
	key(tv, keyboard.KeyBackspace)
	if got := tv.Text(); got != "hel" {
		t.Fatalf("Text = %q, want %q", got, "hel")
	}
}

func TestTextViewDeleteForwardJoins(t *testing.T) {
	tv := newTV(5, "abc", "def")
	seek(tv, 0, 3)
	key(tv, keyboard.KeyDelete)
	if got := tv.Text(); got != "abcdef" {
		t.Fatalf("Text = %q, want %q", got, "abcdef")
	}
}

func TestTextViewCursorMovementClampsColumn(t *testing.T) {
	tv := newTV(5, "longline", "hi", "another")
	seek(tv, 0, 8)            // end of "longline"
	key(tv, keyboard.KeyDown) // onto "hi" (len 2) -> col clamps to 2
	if tv.curLine != 1 || tv.curCol != 2 {
		t.Errorf("after Down: (%d,%d), want (1,2)", tv.curLine, tv.curCol)
	}
	key(tv, keyboard.KeyHome)
	if tv.curCol != 0 {
		t.Errorf("Home: curCol = %d, want 0", tv.curCol)
	}
	key(tv, keyboard.KeyEnd)
	if tv.curCol != 2 {
		t.Errorf("End: curCol = %d, want 2", tv.curCol)
	}
}

func TestTextViewLeftWrapsToPrevLine(t *testing.T) {
	tv := newTV(5, "ab", "cd")
	seek(tv, 1, 0)
	key(tv, keyboard.KeyLeft)
	if tv.curLine != 0 || tv.curCol != 2 {
		t.Errorf("Left wrap: (%d,%d), want (0,2)", tv.curLine, tv.curCol)
	}
}

func TestTextViewUnicodeEditing(t *testing.T) {
	tv := newTV(5)
	typeRunes(tv, "äöü")
	if tv.curCol != 3 {
		t.Errorf("curCol = %d, want 3 (runes, not bytes)", tv.curCol)
	}
	key(tv, keyboard.KeyBackspace)
	if got := tv.Text(); got != "äö" {
		t.Errorf("Text = %q, want %q", got, "äö")
	}
}

func TestTextViewReadOnlyIgnoresEdits(t *testing.T) {
	tv := newTV(5, "fixed")
	tv.ReadOnly = true
	typeRunes(tv, "xyz")
	key(tv, keyboard.KeyBackspace)
	if got := tv.Text(); got != "fixed" {
		t.Errorf("ReadOnly Text = %q, want %q", got, "fixed")
	}
}

func TestTextViewScrollFollowsCursor(t *testing.T) {
	// 5 visible lines, 20 lines of content.
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}
	tv := newTV(5, lines...)
	if tv.VisibleLines() != 5 {
		t.Fatalf("VisibleLines = %d, want 5", tv.VisibleLines())
	}
	// move down past the bottom; top must follow
	for i := 0; i < 10; i++ {
		key(tv, keyboard.KeyDown)
	}
	if tv.curLine != 10 {
		t.Fatalf("curLine = %d, want 10", tv.curLine)
	}
	if tv.TopLine() != 6 { // cursor 10 visible in [6,10]
		t.Errorf("TopLine = %d, want 6", tv.TopLine())
	}
	// back to top
	for i := 0; i < 10; i++ {
		key(tv, keyboard.KeyUp)
	}
	if tv.TopLine() != 0 {
		t.Errorf("TopLine after scroll-up = %d, want 0", tv.TopLine())
	}
}

func TestTextViewOnChangeFires(t *testing.T) {
	tv := newTV(5)
	n := 0
	tv.OnChange = func() { n++ }
	typeRunes(tv, "ab")
	if n != 2 {
		t.Errorf("OnChange fired %d times, want 2", n)
	}
}

// selectRange sets a mouse-style selection from (al,ac) to (cl,cc).
func selectRange(tv *TextView, al, ac, cl, cc int) {
	tv.anchorLine, tv.anchorCol = al, ac
	tv.curLine, tv.curCol = cl, cc
}

func TestTextViewSelectedTextSingleLine(t *testing.T) {
	tv := newTV(5, "hello world")
	selectRange(tv, 0, 0, 0, 5)
	if got := tv.SelectedText(); got != "hello" {
		t.Errorf("SelectedText = %q, want %q", got, "hello")
	}
}

func TestTextViewSelectedTextReversed(t *testing.T) {
	// anchor after cursor: range must normalise.
	tv := newTV(5, "hello world")
	selectRange(tv, 0, 11, 0, 6)
	if got := tv.SelectedText(); got != "world" {
		t.Errorf("SelectedText = %q, want %q", got, "world")
	}
}

func TestTextViewSelectedTextMultiLine(t *testing.T) {
	tv := newTV(5, "abc", "def", "ghi")
	selectRange(tv, 0, 1, 2, 2) // from "bc" .. through .. "gh"
	if got := tv.SelectedText(); got != "bc\ndef\ngh" {
		t.Errorf("SelectedText = %q, want %q", got, "bc\ndef\ngh")
	}
}

func TestTextViewDeleteSelectionMultiLine(t *testing.T) {
	tv := newTV(5, "abc", "def", "ghi")
	selectRange(tv, 0, 1, 2, 2)
	tv.DeleteSelection()
	if got := tv.Text(); got != "ai" {
		t.Errorf("after delete: %q, want %q", got, "ai")
	}
	if tv.curLine != 0 || tv.curCol != 1 {
		t.Errorf("cursor = (%d,%d), want (0,1)", tv.curLine, tv.curCol)
	}
	if tv.hasSelection() {
		t.Error("selection should be collapsed after delete")
	}
}

func TestTextViewTypeOverSelection(t *testing.T) {
	tv := newTV(5, "hello world")
	selectRange(tv, 0, 0, 0, 5) // select "hello"
	tv.edit(keyboard.Event{Rune: 'X'})
	if got := tv.Text(); got != "X world" {
		t.Errorf("type-over: %q, want %q", got, "X world")
	}
}

func TestTextViewBackspaceDeletesSelection(t *testing.T) {
	tv := newTV(5, "hello world")
	selectRange(tv, 0, 0, 0, 6) // "hello "
	tv.edit(keyboard.Event{Key: keyboard.KeyBackspace})
	if got := tv.Text(); got != "world" {
		t.Errorf("backspace selection: %q, want %q", got, "world")
	}
}

func TestTextViewInsertPaste(t *testing.T) {
	tv := newTV(5, "ad")
	seek(tv, 0, 1)
	tv.Insert("bc")
	if got := tv.Text(); got != "abcd" {
		t.Errorf("paste: %q, want %q", got, "abcd")
	}
	// multi-line paste
	tv2 := newTV(5, "")
	tv2.Insert("one\ntwo")
	if got := tv2.Text(); got != "one\ntwo" {
		t.Errorf("multiline paste: %q, want %q", got, "one\ntwo")
	}
}

func TestTextViewSelSpan(t *testing.T) {
	tv := newTV(5, "abcdef", "ghijkl")
	selectRange(tv, 0, 2, 1, 3)
	if s0, s1, ok := tv.selSpan(0); !ok || s0 != 2 || s1 != 6 {
		t.Errorf("line0 span = (%d,%d,%v), want (2,6,true)", s0, s1, ok)
	}
	if s0, s1, ok := tv.selSpan(1); !ok || s0 != 0 || s1 != 3 {
		t.Errorf("line1 span = (%d,%d,%v), want (0,3,true)", s0, s1, ok)
	}
	// no selection -> no span
	tv.collapseSelection()
	if _, _, ok := tv.selSpan(0); ok {
		t.Error("collapsed selection should have no span")
	}
}

func TestTextViewNavigationCollapsesSelection(t *testing.T) {
	tv := newTV(5, "hello")
	selectRange(tv, 0, 0, 0, 5)
	tv.edit(keyboard.Event{Key: keyboard.KeyLeft})
	if tv.hasSelection() {
		t.Error("arrow key should collapse the selection")
	}
}

func TestTextViewSetTextClearsSelection(t *testing.T) {
	tv := newTV(5, "hello world")
	selectRange(tv, 0, 0, 0, 5) // select "hello"
	if !tv.hasSelection() {
		t.Fatal("precondition: expected a selection")
	}
	tv.lines = []string{""} // avoid Draw (no server); exercise the collapse path
	tv.curLine, tv.curCol, tv.top = 0, 0, 0
	tv.collapseSelection()
	if tv.hasSelection() {
		t.Error("SetText/collapse must clear the selection")
	}
}

func TestTextViewExpandTabs(t *testing.T) {
	tv := newTV(5)
	tv.TabWidth = 8
	cases := []struct{ in, want string }{
		{"\tx", "        x"},                 // tab at col 0 -> 8 spaces
		{"ab\tc", "ab      c"},               // tab from col 2 -> 6 spaces (to col 8)
		{"abcdefgh\tx", "abcdefgh        x"}, // tab at col 8 -> next stop col 16
		{"no tabs", "no tabs"},
	}
	for _, c := range cases {
		if got := tv.expand(c.in, -1); got != c.want {
			t.Errorf("expand(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// prefix expansion stops at n runes (n counts buffer runes, not columns)
	if got := tv.expand("\tabc", 1); got != "        " {
		t.Errorf("expand prefix of just the tab = %q, want 8 spaces", got)
	}
}

func TestTextViewExpandKeepsBufferIntact(t *testing.T) {
	// the buffer keeps the tab; only display/metrics expand it.
	tv := newTV(5, "\tindented")
	if tv.Text() != "\tindented" {
		t.Errorf("buffer = %q, want a literal tab preserved", tv.Text())
	}
}

func TestTextViewTabWidthDefault(t *testing.T) {
	tv := newTV(5) // TabWidth 0 -> default 8
	if got := tv.expand("\t.", -1); got != "        ." {
		t.Errorf("default tab width: %q", got)
	}
}
