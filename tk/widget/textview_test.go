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
	tv.curCol = 3
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
	tv.curLine, tv.curCol = 1, 0
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
	tv.curCol = 5
	key(tv, keyboard.KeyBackspace)
	key(tv, keyboard.KeyBackspace)
	if got := tv.Text(); got != "hel" {
		t.Fatalf("Text = %q, want %q", got, "hel")
	}
}

func TestTextViewDeleteForwardJoins(t *testing.T) {
	tv := newTV(5, "abc", "def")
	tv.curLine, tv.curCol = 0, 3
	key(tv, keyboard.KeyDelete)
	if got := tv.Text(); got != "abcdef" {
		t.Fatalf("Text = %q, want %q", got, "abcdef")
	}
}

func TestTextViewCursorMovementClampsColumn(t *testing.T) {
	tv := newTV(5, "longline", "hi", "another")
	tv.curLine, tv.curCol = 0, 8 // end of "longline"
	key(tv, keyboard.KeyDown)    // onto "hi" (len 2) -> col clamps to 2
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
	tv.curLine, tv.curCol = 1, 0
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
