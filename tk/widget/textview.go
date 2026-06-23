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

	gc      *tk_core.GC
	lines   []string
	curLine int // cursor line index
	curCol  int // cursor column as a rune index within the line
	top     int // first visible line
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
	t.EventMask |= base.CARD32(event_mask.KeyPress | event_mask.ButtonPress |
		event_mask.ButtonRelease | event_mask.Exposure)

	t.Window.SetWindowHandler(t)
	if err := t.Window.Create(); err != nil {
		return err
	}

	gc, err := t.Conn.CreateGC1(t.Fg, t.Bg, t.Font.ID)
	if err != nil {
		return err
	}
	t.gc = gc

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
		if t.lines[ln] != "" {
			if err := t.Font.DrawText(t.Drawable, t.gc.XID, 2, base.INT16(i*h), 0, t.lines[ln]); err != nil {
				return err
			}
		}
	}
	return t.drawCaret()
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
		t.focus()
		t.placeCursor(int(e.EventX), int(e.EventY))
		_ = t.Draw()
	case *events.KeyPressEvent:
		if t.Keymap != nil && t.edit(t.Keymap.Lookup(e.Key, e.State)) {
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
		t.insertNewline()
	case keyboard.KeyBackspace:
		t.backspace()
	case keyboard.KeyDelete:
		t.deleteForward()
	case keyboard.KeyNone:
		if k.Printable() {
			t.insertRune(k.Rune)
		} else {
			return false // modifier-only or unhandled key: no repaint
		}
	default:
		return false
	}
	t.ensureVisible()
	return true
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
