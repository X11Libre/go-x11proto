package term

import (
	"fmt"
	"strings"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
)

// X11 core protocol modifier bits, as they appear in a button/motion event's
// State field (same layout tk/keyboard decodes for key events, duplicated
// here rather than exported since it's three well-known protocol constants,
// not shared logic).
const (
	modShift   base.CARD16 = 1 << 0
	modControl base.CARD16 = 1 << 2
	modAlt     base.CARD16 = 1 << 3
)

// Mouse button numbers as the xterm mouse protocol encodes them (0-based;
// distinct from X11's own 1-based button numbering in ButtonPressEvent.Key).
const (
	mouseLeft      = 0
	mouseMiddle    = 1
	mouseRight     = 2
	mouseWheelUp   = 64
	mouseWheelDown = 65
)

// mouseButtonFor converts an X11 button number (ButtonPressEvent.Key: 1=left,
// 2=middle, 3=right, 4=wheel up, 5=wheel down) to the protocol's own
// numbering, reporting ok=false for anything else (buttons 6/7, horizontal
// wheel, aren't reported).
func mouseButtonFor(x11Button base.CARD8) (btn int, ok bool) {
	switch x11Button {
	case 1:
		return mouseLeft, true
	case 2:
		return mouseMiddle, true
	case 3:
		return mouseRight, true
	case btnWheelUp:
		return mouseWheelUp, true
	case btnWheelDown:
		return mouseWheelDown, true
	}
	return 0, false
}

// encodeMouse builds the xterm mouse-protocol byte sequence reporting button
// at 1-based (col, row): SGR (1006) encoding if sgr is set, else legacy X10.
// press is ignored (always encoded as a press) for wheel buttons, which
// xterm never reports a release for. motion marks a drag-tracking report
// (mode 1002/1003) rather than a plain click.
func encodeMouse(button, col, row int, press, motion, shift, alt, ctrl, sgr bool) []byte {
	cb := button
	if motion {
		cb |= 32
	}
	if shift {
		cb |= 4
	}
	if alt {
		cb |= 8
	}
	if ctrl {
		cb |= 16
	}
	if sgr {
		final := byte('M')
		if !press && button != mouseWheelUp && button != mouseWheelDown {
			final = 'm'
		}
		return []byte(fmt.Sprintf("\x1b[<%d;%d;%d%c", cb, col, row, final))
	}
	// Legacy X10: one byte per field, offset by 32; 223 (0xff-32) is the
	// largest column/row it can represent at all, so clamp rather than wrap
	// into garbage on a wide/tall window.
	if col > 223 {
		col = 223
	}
	if row > 223 {
		row = 223
	}
	if !press && button != mouseWheelUp && button != mouseWheelDown {
		cb = 3 // X10 has no notion of which button was released
	}
	return []byte{0x1b, '[', 'M', byte(cb + 32), byte(col + 32), byte(row + 32)}
}

// cellAt converts pixel coordinates to a 0-based (row, col) grid position,
// clamped to the current grid bounds.
func (t *Term) cellAt(x, y base.CARD16) (row, col int) {
	var cw, h int
	if t.AAFace != nil {
		cw, h = t.AAFace.Advance(' '), t.AAFace.Height()
	} else {
		cw, h = t.Font.RuneWidth(' '), t.Font.Height()
	}
	col, row = int(x)/cw, int(y)/h
	if col < 0 {
		col = 0
	} else if col >= t.cols {
		col = t.cols - 1
	}
	if row < 0 {
		row = 0
	} else if row >= t.rows {
		row = t.rows - 1
	}
	return
}

// sendMouseReport encodes and writes one mouse event to the PTY, in whatever
// tracking mode/encoding the running application last negotiated.
func (t *Term) sendMouseReport(button int, x, y base.CARD16, press, motion bool, state base.CARD16) {
	if t.pty == nil {
		return
	}
	t.mu.Lock()
	sgr := t.parser.Modes.MouseSGR
	t.mu.Unlock()
	row, col := t.cellAt(x, y)
	b := encodeMouse(button, col+1, row+1, press, motion,
		state&modShift != 0, state&modAlt != 0, state&modControl != 0, sgr)
	_, _ = t.pty.Master.Write(b)
}

// mouseReportMode returns the currently active tracking mode (0/1000/1002/
// 1003), synchronized against the parser goroutine.
func (t *Term) mouseReportMode() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.parser.Modes.MouseReport
}

// handleButtonPress dispatches a button press either to the running
// application (as a mouse report, if it asked for mouse tracking) or to
// local behaviour: wheel scrolls scrollback, left starts a text selection,
// middle pastes the PRIMARY selection.
func (t *Term) handleButtonPress(e *events.ButtonPressEvent) {
	if btn, ok := mouseButtonFor(e.Key); ok && t.mouseReportMode() != 0 {
		t.sendMouseReport(btn, e.EventX, e.EventY, true, false, e.State)
		return
	}
	switch e.Key {
	case btnWheelUp:
		t.ScrollBy(wheelStepLines)
	case btnWheelDown:
		t.ScrollBy(-wheelStepLines)
	case 1: // left: (re)start a selection; a plain click with no drag clears it
		if t.scrollOffset != 0 {
			return // selection only supports the live view, not scrollback
		}
		row, col := t.cellAt(e.EventX, e.EventY)
		t.selActive = false
		t.selDragging = true
		t.selAnchor = cellPos{row, col}
		t.selCursor = t.selAnchor
		_ = t.Draw()
	case 2: // middle: paste PRIMARY
		if t.primary != nil {
			_, _ = t.primary.RequestText()
		}
	}
}

// handleButtonRelease mirrors handleButtonPress: a mouse report if the
// application wants one, otherwise finalizing a selection drag by taking
// PRIMARY-selection ownership of the selected text.
func (t *Term) handleButtonRelease(e *events.ButtonReleaseEvent) {
	if btn, ok := mouseButtonFor(e.Key); ok && t.mouseReportMode() != 0 {
		t.sendMouseReport(btn, e.EventX, e.EventY, false, false, e.State)
		return
	}
	t.selDragging = false
	if e.Key == 1 && t.selActive && t.primary != nil {
		if text := t.selectedText(); text != "" {
			_ = t.primary.Own(text)
		} else {
			t.selActive = false
		}
	}
}

// handleMotion reports drag motion to the application in 1002/1003 tracking
// mode, or otherwise extends an in-progress local selection.
//
// Limitation: a ButtonMotion event doesn't say which button is held, so a
// reported drag is always attributed to the left button — undercounts a
// middle/right-button drag, but matches what most applications actually
// care about (that a drag happened at all).
func (t *Term) handleMotion(e *events.MotionEvent) {
	if mode := t.mouseReportMode(); mode == 1002 || mode == 1003 {
		t.sendMouseReport(mouseLeft, e.EventX, e.EventY, true, true, e.State)
		return
	}
	if !t.selDragging {
		return
	}
	row, col := t.cellAt(e.EventX, e.EventY)
	if next := (cellPos{row, col}); next != t.selCursor {
		t.selCursor = next
		t.selActive = true
		_ = t.Draw()
	}
}

// linIndex linearizes a cell position for range comparison (row-major,
// i.e. "stream" selection like every mainstream terminal uses — the whole
// tail of the anchor row, all of the rows in between, and the head of the
// cursor row, not a rectangular block).
func linIndex(p cellPos, cols int) int { return p.row*cols + p.col }

// selectedAt reports whether (row, col) falls within the current selection.
func (t *Term) selectedAt(row, col int) bool {
	if !t.selActive {
		return false
	}
	a, b := linIndex(t.selAnchor, t.cols), linIndex(t.selCursor, t.cols)
	if a > b {
		a, b = b, a
	}
	p := linIndex(cellPos{row, col}, t.cols)
	return p >= a && p <= b
}

// selectedText extracts the current selection from the live grid as plain
// text, one line per selected row, each right-trimmed of the blank cells a
// terminal grid pads every row out to.
func (t *Term) selectedText() string {
	if !t.selActive {
		return ""
	}
	a, b := t.selAnchor, t.selCursor
	if linIndex(a, t.cols) > linIndex(b, t.cols) {
		a, b = b, a
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	var lines []string
	for r := a.row; r <= b.row && r < t.grid.Rows; r++ {
		row := t.grid.cur[r]
		start, end := 0, len(row)-1
		if r == a.row {
			start = a.col
		}
		if r == b.row && b.col < end {
			end = b.col
		}
		var sb strings.Builder
		for c := start; c <= end; c++ {
			rn := row[c].Rune
			if rn == 0 {
				rn = ' '
			}
			sb.WriteRune(rn)
		}
		lines = append(lines, strings.TrimRight(sb.String(), " "))
	}
	return strings.Join(lines, "\n")
}
