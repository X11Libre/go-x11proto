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
	TabWidth    int         // tab stop in columns (default 8)

	gc    *tk_core.GC
	selGc *tk_core.GC
	lines []string

	curLine int // cursor line index
	curCol  int // cursor column as a rune index within the line

	// selection anchor; the active end is the cursor. Empty when equal to it.
	anchorLine int
	anchorCol  int
	selecting  bool

	top    int // first visible line
	leftPx int // horizontal scroll offset in pixels

	undo, redo []tvSnapshot
	recording  bool // an undo snapshot has been taken for the current action
}

// tvSnapshot is an undo/redo entry: a copy of the buffer plus the cursor.
type tvSnapshot struct {
	lines     []string
	line, col int
}

const undoLimit = 500

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
		event_mask.ButtonRelease | event_mask.Button1Motion | event_mask.Exposure |
		event_mask.StructureNotify)

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
	t.curLine, t.curCol, t.top, t.leftPx = 0, 0, 0, 0
	t.collapseSelection() // a fresh buffer has no selection
	t.undo, t.redo = nil, nil
	t.changed()
	t.scrolled()
	_ = t.Draw()
}

// Text returns the buffer contents joined with newlines.
func (t *TextView) Text() string { return strings.Join(t.lines, "\n") }

// --- tab handling -------------------------------------------------------
//
// Tabs are kept verbatim in the buffer but expanded to the next tab stop for
// display and metrics, so tab-indented text (i.e. most code) lines up instead
// of collapsing to the left.

func (t *TextView) tabCols() int {
	if t.TabWidth > 0 {
		return t.TabWidth
	}
	return 8
}

// expand returns the first n runes of line (n < 0 = all) with tabs replaced by
// spaces up to each tab stop, as drawn on screen.
func (t *TextView) expand(line string, n int) string {
	tw := t.tabCols()
	var b strings.Builder
	col := 0
	for i, r := range []rune(line) {
		if n >= 0 && i >= n {
			break
		}
		if r == '\t' {
			for s := tw - col%tw; s > 0; s-- {
				b.WriteByte(' ')
				col++
			}
		} else {
			b.WriteRune(r)
			col++
		}
	}
	return b.String()
}

// originX is the on-screen x of column 0 (left margin minus horizontal scroll).
func (t *TextView) originX() int { return 2 - t.leftPx }

// colX is the pixel x (from the text origin) of rune column col on line.
func (t *TextView) colX(line string, col int) int {
	return t.Font.TextWidth(t.expand(line, col))
}

// colAtX maps a pixel offset (from the text origin) to the nearest rune column,
// accounting for tab expansion.
func (t *TextView) colAtX(line string, px int) int {
	if px <= 0 {
		return 0
	}
	tw := t.tabCols()
	sp := t.Font.RuneWidth(' ')
	rs := []rune(line)
	col, acc := 0, 0
	for i, r := range rs {
		var w int
		if r == '\t' {
			n := tw - col%tw
			w = n * sp
			col += n
		} else {
			w = t.Font.RuneWidth(r)
			col++
		}
		if px < acc+w/2+1 {
			return i
		}
		acc += w
	}
	return len(rs)
}

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
	if t.gc == nil {
		return nil // not realised yet (e.g. offline unit tests)
	}
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
		ox := t.originX()
		if s0, s1, ok := t.selSpan(ln); ok {
			x0 := ox + t.colX(t.lines[ln], s0)
			x1 := ox + t.colX(t.lines[ln], s1)
			if err := t.FillRect(t.selGc.XID, base.INT16(x0), y, base.CARD16(x1-x0), base.CARD16(h)); err != nil {
				return err
			}
		}
		if t.lines[ln] != "" {
			if err := t.Font.DrawText(t.Drawable, t.gc.XID, base.INT16(ox), y, 0, t.expand(t.lines[ln], -1)); err != nil {
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
	x := t.originX() + t.colX(t.lines[t.curLine], t.curCol)
	y := row * t.Font.Height()
	return t.FillRect(t.gc.XID, base.INT16(x), base.INT16(y), 1, base.CARD16(t.Font.Height()))
}

// HandleWindowEvent drives drawing, editing and pointer focus.
func (t *TextView) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		_ = t.Draw()
	case *events.ConfigureEvent:
		t.W, t.H = e.Width, e.Height
		t.ensureVisible()
		_ = t.Draw()
	case *events.ButtonPressEvent:
		switch e.Key {
		case 1: // left button: place cursor and begin selecting
			t.Focus()
			t.placeCursor(int(e.EventX), int(e.EventY))
			t.collapseSelection()
			t.selecting = true
			_ = t.Draw()
		case btnWheelUp: // touchpad / wheel scroll up
			t.ScrollTo(t.top - wheelStepLines)
		case btnWheelDown:
			t.ScrollTo(t.top + wheelStepLines)
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

// Focus requests the keyboard focus so typing reaches this widget. It is called
// automatically on a button press, but an app can also call it directly (e.g.
// after dismissing a dialog).
func (t *TextView) Focus() {
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
	t.curCol = t.colAtX(t.lines[row], x-t.originX())
}

// edit applies one decoded key event to the buffer/cursor and reports whether a
// repaint is needed. It does no drawing itself, so it is testable without a
// server.
func (t *TextView) edit(k keyboard.Event) bool {
	hadSel := t.hasSelection()
	nav := false // a pure cursor move (Shift extends, otherwise collapses)
	mutating := keyMutates(k)
	var pushed bool
	if mutating {
		pushed = t.beginEdit()
	}

	switch k.Key {
	case keyboard.KeyLeft:
		t.moveLeft()
		nav = true
	case keyboard.KeyRight:
		t.moveRight()
		nav = true
	case keyboard.KeyUp:
		t.moveVert(-1)
		nav = true
	case keyboard.KeyDown:
		t.moveVert(1)
		nav = true
	case keyboard.KeyHome:
		t.curCol = 0
		nav = true
	case keyboard.KeyEnd:
		t.curCol = t.runeLen(t.curLine)
		nav = true
	case keyboard.KeyPageUp:
		t.moveVert(-t.VisibleLines())
		nav = true
	case keyboard.KeyPageDown:
		t.moveVert(t.VisibleLines())
		nav = true
	case keyboard.KeyTab:
		t.replaceSelection()
		t.insertRune('\t')
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
			t.endEdit(pushed)
			return false // modifier-only or unhandled key: no repaint
		}
	default:
		t.endEdit(pushed)
		return false
	}

	// Shift+navigation extends the selection; everything else collapses it.
	if !nav || !k.Shift {
		t.collapseSelection()
	}
	t.endEdit(pushed)
	t.ensureVisible()
	return true
}

// keyMutates reports whether a key changes the buffer (and so needs an undo
// snapshot).
func keyMutates(k keyboard.Event) bool {
	switch k.Key {
	case keyboard.KeyTab, keyboard.KeyEnter, keyboard.KeyBackspace, keyboard.KeyDelete:
		return true
	case keyboard.KeyNone:
		return k.Printable()
	}
	return false
}

// SelectAll selects the whole buffer.
func (t *TextView) SelectAll() {
	t.anchorLine, t.anchorCol = 0, 0
	t.curLine = len(t.lines) - 1
	t.curCol = t.runeLen(t.curLine)
	t.ensureVisible()
	_ = t.Draw()
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
	pushed := t.beginEdit()
	defer t.endEdit(pushed)
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
	pushed := t.beginEdit()
	defer t.endEdit(pushed)
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

// --- search & replace ---

// FindNext selects the next occurrence of query at or after the cursor and
// scrolls to it; the search wraps around the buffer. Returns false if there is
// no match (or query is empty).
func (t *TextView) FindNext(query string) bool {
	l, c, ok := searchFrom(t.lines, query, t.curLine, t.curCol+1)
	if !ok {
		return false
	}
	t.anchorLine, t.anchorCol = l, c
	t.curLine, t.curCol = l, c+len([]rune(query))
	t.ensureVisible()
	_ = t.Draw()
	return true
}

// ReplaceAll replaces every occurrence of old with new and returns the count.
func (t *TextView) ReplaceAll(old, new string) int {
	if old == "" || t.ReadOnly {
		return 0
	}
	n := 0
	for _, ln := range t.lines {
		n += strings.Count(ln, old)
	}
	if n == 0 {
		return 0
	}
	pushed := t.beginEdit()
	defer t.endEdit(pushed)
	for i := range t.lines {
		t.lines[i] = strings.ReplaceAll(t.lines[i], old, new)
	}
	t.curLine, t.curCol, t.top = 0, 0, 0
	t.collapseSelection()
	t.changed()
	_ = t.Draw()
	return n
}

// searchFrom finds query at or after (fromLine, fromCol), wrapping once around
// the buffer. Positions are rune indices. Matching is within a single line.
func searchFrom(lines []string, query string, fromLine, fromCol int) (line, col int, ok bool) {
	if query == "" || len(lines) == 0 {
		return 0, 0, false
	}
	n := len(lines)
	if fromLine < 0 || fromLine >= n {
		fromLine = 0
	}
	for d := 0; d < n; d++ {
		li := (fromLine + d) % n
		rs := []rune(lines[li])
		from := 0
		if d == 0 {
			from = clampInt(fromCol, 0, len(rs))
		}
		if idx := runeIndex(string(rs[from:]), query); idx >= 0 {
			return li, from + idx, true
		}
	}
	// wrap: the head of the starting line (before fromCol) was not searched above
	if idx := runeIndex(lines[fromLine], query); idx >= 0 {
		return fromLine, idx, true
	}
	return 0, 0, false
}

// runeIndex returns the rune index of the first occurrence of q in s, or -1.
func runeIndex(s, q string) int {
	b := strings.Index(s, q)
	if b < 0 {
		return -1
	}
	return len([]rune(s[:b]))
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

// ensureVisible scrolls vertically and horizontally so the cursor is on screen.
func (t *TextView) ensureVisible() {
	if t.curLine < t.top {
		t.top = t.curLine
		t.scrolled()
	} else if vis := t.VisibleLines(); t.curLine >= t.top+vis {
		t.top = t.curLine - vis + 1
		t.scrolled()
	}
	t.ensureColVisible()
}

// ensureColVisible adjusts the horizontal offset so the caret is within the
// visible width.
func (t *TextView) ensureColVisible() {
	caret := t.colX(t.lines[t.curLine], t.curCol)
	view := int(t.W) - 4 // text area minus the left/right margin
	if view < 1 {
		view = 1
	}
	const pad = 8 // keep a little context past the caret
	if caret < t.leftPx {
		t.leftPx = max0(caret - pad)
	} else if caret > t.leftPx+view {
		t.leftPx = caret - view + pad
	}
	if t.leftPx < 0 {
		t.leftPx = 0
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

// --- undo / redo ---
//
// Each mutating action snapshots the buffer+cursor before changing it. A
// reentrancy guard (recording) keeps a compound action (e.g. replace selection
// then insert) to a single undo step.

func (t *TextView) snapshot() tvSnapshot {
	cp := make([]string, len(t.lines))
	copy(cp, t.lines)
	return tvSnapshot{lines: cp, line: t.curLine, col: t.curCol}
}

// beginEdit takes an undo snapshot unless one was already taken for the current
// action; it returns true when it was the outermost call.
func (t *TextView) beginEdit() bool {
	if t.recording {
		return false
	}
	t.recording = true
	t.undo = append(t.undo, t.snapshot())
	if len(t.undo) > undoLimit {
		t.undo = t.undo[1:]
	}
	t.redo = nil
	return true
}

func (t *TextView) endEdit(pushed bool) {
	if pushed {
		t.recording = false
	}
}

func (t *TextView) restore(s tvSnapshot) {
	t.lines = s.lines
	if len(t.lines) == 0 {
		t.lines = []string{""}
	}
	t.curLine = clampInt(s.line, 0, len(t.lines)-1)
	t.curCol = clampInt(s.col, 0, t.runeLen(t.curLine))
	t.collapseSelection()
}

// Undo reverts the last edit.
func (t *TextView) Undo() {
	if len(t.undo) == 0 {
		return
	}
	t.redo = append(t.redo, t.snapshot())
	s := t.undo[len(t.undo)-1]
	t.undo = t.undo[:len(t.undo)-1]
	t.restore(s)
	t.changed()
	t.ensureVisible()
	_ = t.Draw()
}

// Redo re-applies the last undone edit.
func (t *TextView) Redo() {
	if len(t.redo) == 0 {
		return
	}
	t.undo = append(t.undo, t.snapshot())
	s := t.redo[len(t.redo)-1]
	t.redo = t.redo[:len(t.redo)-1]
	t.restore(s)
	t.changed()
	t.ensureVisible()
	_ = t.Draw()
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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
