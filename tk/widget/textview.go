package widget

import (
	"strings"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// TextView is a multi-line, editable text area: a buffer of lines drawn with a
// core font, a blinking-less block/bar caret, keyboard editing and navigation,
// click-to-position, and vertical scrolling. It is the heart of an xedit-style
// editor.
//
// Fill in the embedded Window (Parent/X/Y/W/H) and Font before Init; Keymap is
// loaded automatically if nil. The caller may set Fg/Bg (default black on
// white) and ReadOnly. OnChange fires after any buffer edit and OnScroll after
// the top line moves, so a status line or scrollbar can refresh.
type TextView struct {
	tk_core.Window
	Font     *font.Font
	Keymap   *keyboard.Map
	Fg, Bg   base.CARD32
	ReadOnly bool

	OnChange func()
	OnScroll func()
	// OnSelect fires when a mouse selection finishes with non-empty text, so the
	// app can own the PRIMARY selection. OnKey, if set, sees each decoded key
	// before default handling; returning true marks it handled (e.g. to bind
	// Ctrl+C/V/X to the clipboard).
	OnSelect func(string)
	OnKey    func(keyboard.Event) bool

	SelectionBg base.CARD32 // highlight colour (default light blue)

	gc    *tk_core.GC
	selGc *tk_core.GC
	lines []string

	curLine int // cursor line index
	curCol  int // cursor column as a rune index within the line

	// selection anchor; the active end is the cursor. Empty when equal to it.
	anchorLine int
	anchorCol  int
	selecting  bool

	top int // first visible line
}

// Init creates and maps the text view, builds its GC, and loads the keyboard
// map. Font must be set first.
func (t *TextView) Init() error {
	if t.Fg == 0 && t.Bg == 0 {
		t.Fg = t.Conn.X11Conn.DefaultBlackPixel()
		t.Bg = t.Conn.X11Conn.DefaultWhitePixel()
	}
	if len(t.lines) == 0 {
		t.lines = []string{""}
	}
	if t.SelectionBg == 0 {
		t.SelectionBg = 0xb0c4ff // light blue
	}
	t.EventMask |= base.CARD32(event_mask.KeyPress | event_mask.ButtonPress |
		event_mask.ButtonRelease | event_mask.Button1Motion | event_mask.Exposure)

	t.Window.SetWindowHandler(t)
	if err := t.Window.Create(); err != nil {
		return err
	}

	gc, err := t.Conn.CreateGC1(t.Fg, t.Bg, t.Font.ID)
	if err != nil {
		return err
	}
	t.gc = gc
	selGc, err := t.Conn.CreateGC1(t.SelectionBg, t.Bg, t.Font.ID)
	if err != nil {
		return err
	}
	t.selGc = selGc

	if t.Keymap == nil {
		if km, err := keyboard.Load(t.Conn.X11Conn); err == nil {
			t.Keymap = km
		}
	}
	return t.Window.Map()
}

// SetText replaces the whole buffer and resets the cursor to the start.
func (t *TextView) SetText(s string) {
	t.lines = strings.Split(s, "\n")
	if len(t.lines) == 0 {
		t.lines = []string{""}
	}
	t.curLine, t.curCol, t.top = 0, 0, 0
	t.changed()
	t.scrolled()
	_ = t.Draw()
}

// Text returns the buffer contents joined with newlines.
func (t *TextView) Text() string { return strings.Join(t.lines, "\n") }

// LineCount / VisibleLines / TopLine expose scroll state for a scrollbar.
func (t *TextView) LineCount() int    { return len(t.lines) }
func (t *TextView) VisibleLines() int { return max1(int(t.H) / t.Font.Height()) }
func (t *TextView) TopLine() int      { return t.top }

// ScrollTo sets the first visible line (clamped) and repaints.
func (t *TextView) ScrollTo(line int) {
	maxTop := max0(len(t.lines) - t.VisibleLines())
	if line < 0 {
		line = 0
	}
	if line > maxTop {
		line = maxTop
	}
	if line != t.top {
		t.top = line
		t.scrolled()
		_ = t.Draw()
	}
}

// Draw repaints the visible lines and the caret.
func (t *TextView) Draw() error {
	if err := t.ClearArea(0, 0, 0, 0, false); err != nil {
		return err
	}
	h := t.Font.Height()
	vis := t.VisibleLines()
	for i := 0; i < vis; i++ {
		ln := t.top + i
		if ln >= len(t.lines) {
			break
		}
		y := base.INT16(i * h)
		if s0, s1, ok := t.selSpan(ln); ok {
			rs := []rune(t.lines[ln])
			x0 := 2 + t.Font.TextWidth(string(rs[:s0]))
			x1 := 2 + t.Font.TextWidth(string(rs[:s1]))
			if err := t.FillRect(t.selGc.XID, base.INT16(x0), y, base.CARD16(x1-x0), base.CARD16(h)); err != nil {
				return err
			}
		}
		if t.lines[ln] != "" {
			if err := t.Font.DrawText(t.Drawable, t.gc.XID, 2, y, 0, t.lines[ln]); err != nil {
				return err
			}
		}
	}
	return t.drawCaret()
}

// selSpan returns the selected rune range [s0,s1) on visible line ln, if any.
func (t *TextView) selSpan(ln int) (s0, s1 int, ok bool) {
	if !t.hasSelection() {
		return 0, 0, false
	}
	l0, c0, l1, c1 := t.selRange()
	if ln < l0 || ln > l1 {
		return 0, 0, false
	}
	n := t.runeLen(ln)
	s0, s1 = 0, n
	if ln == l0 {
		s0 = c0
	}
	if ln == l1 {
		s1 = c1
	}
	if s1 < s0 {
		s1 = s0
	}
	return s0, s1, s1 > s0
}

// drawCaret paints a vertical bar at the cursor, if it is on a visible line.
func (t *TextView) drawCaret() error {
	if t.ReadOnly {
		return nil
	}
	row := t.curLine - t.top
	if row < 0 || row >= t.VisibleLines() {
		return nil
	}
	line := t.lines[t.curLine]
	x := 2 + t.Font.TextWidth(string([]rune(line)[:t.curCol]))
	y := row * t.Font.Height()
	return t.FillRect(t.gc.XID, base.INT16(x), base.INT16(y), 1, base.CARD16(t.Font.Height()))
}

// HandleWindowEvent drives drawing, editing and pointer focus.
func (t *TextView) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		_ = t.Draw()
	case *events.ButtonPressEvent:
		if e.Key == 1 { // left button: place cursor and begin selecting
			t.focus()
			t.placeCursor(int(e.EventX), int(e.EventY))
			t.collapseSelection()
			t.selecting = true
			_ = t.Draw()
		}
	case *events.MotionEvent:
		if t.selecting {
			t.placeCursor(int(e.EventX), int(e.EventY))
			_ = t.Draw()
		}
	case *events.ButtonReleaseEvent:
		if t.selecting {
			t.selecting = false
			if t.hasSelection() && t.OnSelect != nil {
				t.OnSelect(t.SelectedText())
			}
		}
	case *events.KeyPressEvent:
		if t.Keymap == nil {
			return true
		}
		k := t.Keymap.Lookup(e.Key, e.State)
		if t.OnKey != nil && t.OnKey(k) {
			return true
		}
		if t.edit(k) {
			_ = t.Draw()
		}
	}
	return true
}

// focus requests the keyboard focus so typing reaches this widget.
func (t *TextView) focus() {
	_ = rpc.SetInputFocus(t.Conn.X11Conn, 2 /*RevertToParent*/, t.XID, 0)
}

// placeCursor moves the cursor to the character nearest pixel (x, y).
func (t *TextView) placeCursor(x, y int) {
	row := t.top + y/t.Font.Height()
	if row < 0 {
		row = 0
	}
	if row >= len(t.lines) {
		row = len(t.lines) - 1
	}
	t.curLine = row
	t.curCol = t.Font.IndexAtX(t.lines[row], x-2)
}

// edit applies one decoded key event to the buffer/cursor and reports whether a
// repaint is needed. It does no drawing itself, so it is testable without a
// server.
func (t *TextView) edit(k keyboard.Event) bool {
	hadSel := t.hasSelection()
	switch k.Key {
	case keyboard.KeyLeft:
		t.moveLeft()
	case keyboard.KeyRight:
		t.moveRight()
	case keyboard.KeyUp:
		t.moveVert(-1)
	case keyboard.KeyDown:
		t.moveVert(1)
	case keyboard.KeyHome:
		t.curCol = 0
	case keyboard.KeyEnd:
		t.curCol = t.runeLen(t.curLine)
	case keyboard.KeyPageUp:
		t.moveVert(-t.VisibleLines())
	case keyboard.KeyPageDown:
		t.moveVert(t.VisibleLines())
	case keyboard.KeyEnter:
		t.replaceSelection()
		t.insertNewline()
	case keyboard.KeyBackspace:
		if hadSel {
			t.DeleteSelection()
		} else {
			t.backspace()
		}
	case keyboard.KeyDelete:
		if hadSel {
			t.DeleteSelection()
		} else {
			t.deleteForward()
		}
	case keyboard.KeyNone:
		if k.Printable() {
			t.replaceSelection()
			t.insertRune(k.Rune)
		} else {
			return false // modifier-only or unhandled key: no repaint
		}
	default:
		return false
	}
	t.collapseSelection() // keyboard navigation/editing collapses the selection
	t.ensureVisible()
	return true
}

// replaceSelection deletes the selection (if any) before an insert.
func (t *TextView) replaceSelection() {
	if t.hasSelection() {
		t.DeleteSelection()
	}
}

// --- selection ---

func (t *TextView) hasSelection() bool {
	return t.anchorLine != t.curLine || t.anchorCol != t.curCol
}

// selRange returns the selection ordered as (l0,c0) <= (l1,c1).
func (t *TextView) selRange() (l0, c0, l1, c1 int) {
	al, ac, cl, cc := t.anchorLine, t.anchorCol, t.curLine, t.curCol
	if al < cl || (al == cl && ac <= cc) {
		return al, ac, cl, cc
	}
	return cl, cc, al, ac
}

func (t *TextView) collapseSelection() {
	t.anchorLine, t.anchorCol = t.curLine, t.curCol
}

// SelectedText returns the currently selected text (empty if no selection).
func (t *TextView) SelectedText() string {
	if !t.hasSelection() {
		return ""
	}
	l0, c0, l1, c1 := t.selRange()
	if l0 == l1 {
		return string([]rune(t.lines[l0])[c0:c1])
	}
	var b strings.Builder
	b.WriteString(string([]rune(t.lines[l0])[c0:]))
	for ln := l0 + 1; ln < l1; ln++ {
		b.WriteByte('\n')
		b.WriteString(t.lines[ln])
	}
	b.WriteByte('\n')
	b.WriteString(string([]rune(t.lines[l1])[:c1]))
	return b.String()
}

// DeleteSelection removes the selected range and places the cursor at its start.
func (t *TextView) DeleteSelection() {
	if t.ReadOnly || !t.hasSelection() {
		return
	}
	l0, c0, l1, c1 := t.selRange()
	head := string([]rune(t.lines[l0])[:c0])
	tail := string([]rune(t.lines[l1])[c1:])
	t.lines[l0] = head + tail
	if l1 > l0 {
		t.lines = append(t.lines[:l0+1], t.lines[l1+1:]...)
	}
	t.curLine, t.curCol = l0, c0
	t.collapseSelection()
	t.changed()
}

// Insert replaces any selection with s (which may contain newlines).
func (t *TextView) Insert(s string) {
	if t.ReadOnly {
		return
	}
	if t.hasSelection() {
		t.DeleteSelection()
	}
	for _, r := range s {
		if r == '\n' {
			t.insertNewline()
		} else {
			t.insertRune(r)
		}
	}
	t.collapseSelection()
	t.ensureVisible()
}

// --- editing primitives (operate on rune slices so multibyte is safe) ---

func (t *TextView) runeLen(line int) int { return len([]rune(t.lines[line])) }

func (t *TextView) insertRune(r rune) {
	if t.ReadOnly {
		return
	}
	rs := []rune(t.lines[t.curLine])
	rs = append(rs[:t.curCol], append([]rune{r}, rs[t.curCol:]...)...)
	t.lines[t.curLine] = string(rs)
	t.curCol++
	t.changed()
}

func (t *TextView) insertNewline() {
	if t.ReadOnly {
		return
	}
	rs := []rune(t.lines[t.curLine])
	head, tail := string(rs[:t.curCol]), string(rs[t.curCol:])
	t.lines[t.curLine] = head
	rest := append([]string{tail}, t.lines[t.curLine+1:]...)
	t.lines = append(t.lines[:t.curLine+1], rest...)
	t.curLine++
	t.curCol = 0
	t.changed()
}

func (t *TextView) backspace() {
	if t.ReadOnly {
		return
	}
	if t.curCol > 0 {
		rs := []rune(t.lines[t.curLine])
		t.lines[t.curLine] = string(append(rs[:t.curCol-1], rs[t.curCol:]...))
		t.curCol--
		t.changed()
		return
	}
	if t.curLine > 0 {
		prev := t.runeLen(t.curLine - 1)
		t.lines[t.curLine-1] += t.lines[t.curLine]
		t.lines = append(t.lines[:t.curLine], t.lines[t.curLine+1:]...)
		t.curLine--
		t.curCol = prev
		t.changed()
	}
}

func (t *TextView) deleteForward() {
	if t.ReadOnly {
		return
	}
	rs := []rune(t.lines[t.curLine])
	if t.curCol < len(rs) {
		t.lines[t.curLine] = string(append(rs[:t.curCol], rs[t.curCol+1:]...))
		t.changed()
		return
	}
	if t.curLine < len(t.lines)-1 {
		t.lines[t.curLine] += t.lines[t.curLine+1]
		t.lines = append(t.lines[:t.curLine+1], t.lines[t.curLine+2:]...)
		t.changed()
	}
}

// --- cursor movement ---

func (t *TextView) moveLeft() {
	if t.curCol > 0 {
		t.curCol--
	} else if t.curLine > 0 {
		t.curLine--
		t.curCol = t.runeLen(t.curLine)
	}
}

func (t *TextView) moveRight() {
	if t.curCol < t.runeLen(t.curLine) {
		t.curCol++
	} else if t.curLine < len(t.lines)-1 {
		t.curLine++
		t.curCol = 0
	}
}

func (t *TextView) moveVert(d int) {
	t.curLine += d
	if t.curLine < 0 {
		t.curLine = 0
	}
	if t.curLine >= len(t.lines) {
		t.curLine = len(t.lines) - 1
	}
	if n := t.runeLen(t.curLine); t.curCol > n {
		t.curCol = n
	}
}

// ensureVisible scrolls so the cursor line is on screen.
func (t *TextView) ensureVisible() {
	if t.curLine < t.top {
		t.top = t.curLine
		t.scrolled()
	} else if vis := t.VisibleLines(); t.curLine >= t.top+vis {
		t.top = t.curLine - vis + 1
		t.scrolled()
	}
}

func (t *TextView) changed() {
	if t.OnChange != nil {
		t.OnChange()
	}
}

func (t *TextView) scrolled() {
	if t.OnScroll != nil {
		t.OnScroll()
	}
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}
