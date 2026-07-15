package term

// Attr is a bitmask of SGR text attributes (colour is carried separately in
// Cell.Fg/Bg).
type Attr uint16

const (
	AttrBold Attr = 1 << iota
	AttrFaint
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrReverse
	AttrConceal
	AttrStrikethrough
)

// Cell is one character position in the grid: a rune plus its style. The
// zero Cell is a blank space in the Term's default colours.
type Cell struct {
	Rune   rune
	Fg, Bg Color
	Attr   Attr
	Link   string // active OSC 8 hyperlink URI for this cell, "" if none
}

// Grid is the fixed Rows x Cols character buffer a Parser mutates and a Term
// draws. It has no notion of "the whole buffer as a string", undo, or
// client-side insert/delete-anywhere editing — everything happens through
// PutRune-at-cursor and the erase/scroll operations a real terminal exposes,
// matching how a byte stream actually drives a VT100-family display.
//
// Grid supports one alternate screen buffer (for full-screen apps like vim
// that DECSET 1049 into it and restore the main screen on exit) and an
// unbounded-until-trimmed scrollback of lines pushed off the top by scrolling
// or output, when the Term's Type.Scrollback is set.
type Grid struct {
	Rows, Cols int
	cur, alt   [][]Cell // cur is whichever of the primary/alt screen is active
	primary    [][]Cell
	altBuf     [][]Cell
	onAlt      bool

	scrollback           [][]Cell
	scrollbackCap        int
	scrollTop, scrollBot int // scrolling region [scrollTop, scrollBot], 0-based inclusive

	CursorRow, CursorCol int
	CursorVisible        bool
	SavedRow, SavedCol   int // DECSC/DECRC

	DefaultFg, DefaultBg Color
}

const defaultScrollbackCap = 10000

// NewGrid creates a Grid of the given size, cursor visible at (0,0).
func NewGrid(rows, cols int) *Grid {
	g := &Grid{
		Rows: rows, Cols: cols,
		CursorVisible: true,
		scrollBot:     rows - 1,
		scrollbackCap: defaultScrollbackCap,
	}
	g.primary = newRows(rows, cols)
	g.altBuf = newRows(rows, cols)
	g.cur = g.primary
	return g
}

func newRows(rows, cols int) [][]Cell {
	rs := make([][]Cell, rows)
	for i := range rs {
		rs[i] = make([]Cell, cols)
	}
	return rs
}

func newRow(cols int) []Cell { return make([]Cell, cols) }

// Cell returns the cell at (row, col), or a blank Cell if out of range.
func (g *Grid) Cell(row, col int) Cell {
	if row < 0 || row >= g.Rows || col < 0 || col >= g.Cols {
		return Cell{}
	}
	return g.cur[row][col]
}

// blank is the Cell erase/scroll fills with: a space in the current default
// colours (matches real terminals — ED/EL paint with the active SGR
// background, not always black).
func (g *Grid) blank() Cell { return Cell{Rune: ' ', Fg: g.DefaultFg, Bg: g.DefaultBg} }

// PutRune writes r at the cursor in the given style (link is an optional OSC 8
// hyperlink URI to tag the cell with) and advances the cursor one column. It
// does not itself wrap: if the cursor is already past the last column it
// clamps back onto it, so repeated writes with no wrap simply keep
// overwriting the last cell. Deciding whether to wrap first is the caller's
// job (see Parser.putRune), since that decision depends on the AutoWrap mode,
// which Grid has no notion of.
func (g *Grid) PutRune(r rune, fg, bg Color, attr Attr, link string) {
	if g.CursorCol >= g.Cols {
		g.CursorCol = g.Cols - 1
	}
	g.cur[g.CursorRow][g.CursorCol] = Cell{Rune: r, Fg: fg, Bg: bg, Attr: attr, Link: link}
	g.CursorCol++
}

// newline moves the cursor to the start of the next line, scrolling the
// scroll region up by one if the cursor was on its last row.
func (g *Grid) newline() {
	if g.CursorRow == g.scrollBot {
		g.ScrollUp(1)
		return
	}
	if g.CursorRow < g.Rows-1 {
		g.CursorRow++
	}
}

// Index (IND) is newline without the column reset LF also does not do.
func (g *Grid) Index() { g.newline() }

// ScrollUp shifts the scroll region [scrollTop, scrollBot] up by n lines,
// pushing the top n lines of the region into scrollback (only on the primary
// screen, and only when scrolling the whole screen — a partial scroll region,
// like a pager's status line, has no scrollback semantics on real terminals
// either) and filling the bottom n with blanks.
func (g *Grid) ScrollUp(n int) {
	top, bot := g.scrollTop, g.scrollBot
	if n <= 0 || top > bot {
		return
	}
	if n > bot-top+1 {
		n = bot - top + 1
	}
	if !g.onAlt && top == 0 && g.scrollbackCap > 0 {
		for i := 0; i < n; i++ {
			g.pushScrollback(g.cur[top+i])
		}
	}
	copy(g.cur[top:bot+1], g.cur[top+n:bot+1])
	for i := bot - n + 1; i <= bot; i++ {
		g.cur[i] = newRow(g.Cols)
		for c := range g.cur[i] {
			g.cur[i][c] = g.blank()
		}
	}
}

// ScrollDown shifts the scroll region down by n lines (used by RI/SU-reverse
// and the CSI T sequence), filling the top n with blanks. It never touches
// scrollback — content scrolled back down was never actually lost.
func (g *Grid) ScrollDown(n int) {
	top, bot := g.scrollTop, g.scrollBot
	if n <= 0 || top > bot {
		return
	}
	if n > bot-top+1 {
		n = bot - top + 1
	}
	copy(g.cur[top+n:bot+1], g.cur[top:bot+1-n])
	for i := top; i < top+n; i++ {
		g.cur[i] = newRow(g.Cols)
		for c := range g.cur[i] {
			g.cur[i][c] = g.blank()
		}
	}
}

func (g *Grid) pushScrollback(row []Cell) {
	cp := make([]Cell, len(row))
	copy(cp, row)
	g.scrollback = append(g.scrollback, cp)
	if len(g.scrollback) > g.scrollbackCap {
		g.scrollback = g.scrollback[len(g.scrollback)-g.scrollbackCap:]
	}
}

// ScrollbackLines returns the n most recent scrollback lines, oldest first.
// Used by Term's ScrollTo/ScrollBy to render a scrolled-back view.
func (g *Grid) ScrollbackLines(n int) [][]Cell {
	if n > len(g.scrollback) {
		n = len(g.scrollback)
	}
	return g.scrollback[len(g.scrollback)-n:]
}

// ScrollbackLen returns how many lines are available via ScrollbackLines.
func (g *Grid) ScrollbackLen() int { return len(g.scrollback) }

// SetScrollRegion sets the DECSTBM scrolling region, 0-based inclusive,
// clamped to the grid and reset to the whole screen if invalid.
func (g *Grid) SetScrollRegion(top, bot int) {
	if top < 0 || bot >= g.Rows || bot < top {
		top, bot = 0, g.Rows-1
	}
	g.scrollTop, g.scrollBot = top, bot
}

// EraseMode selects what ED/EL erase relative to the cursor.
type EraseMode int

const (
	EraseToEnd EraseMode = iota
	EraseToStart
	EraseAll
)

// EraseLine clears part or all of the cursor's row.
func (g *Grid) EraseLine(mode EraseMode) {
	row := g.cur[g.CursorRow]
	from, to := 0, g.Cols-1
	switch mode {
	case EraseToEnd:
		from = g.CursorCol
	case EraseToStart:
		to = g.CursorCol
	}
	for c := from; c <= to && c < g.Cols; c++ {
		row[c] = g.blank()
	}
}

// EraseDisplay clears part or all of the screen.
func (g *Grid) EraseDisplay(mode EraseMode) {
	switch mode {
	case EraseToEnd:
		g.EraseLine(EraseToEnd)
		for r := g.CursorRow + 1; r < g.Rows; r++ {
			g.clearRow(r)
		}
	case EraseToStart:
		g.EraseLine(EraseToStart)
		for r := 0; r < g.CursorRow; r++ {
			g.clearRow(r)
		}
	case EraseAll:
		for r := 0; r < g.Rows; r++ {
			g.clearRow(r)
		}
	}
}

func (g *Grid) clearRow(r int) {
	for c := 0; c < g.Cols; c++ {
		g.cur[r][c] = g.blank()
	}
}

// SetCursor moves the cursor to (row, col), clamped to the grid.
func (g *Grid) SetCursor(row, col int) {
	g.CursorRow = clamp(row, 0, g.Rows-1)
	g.CursorCol = clamp(col, 0, g.Cols-1)
}

// SaveCursor/RestoreCursor implement DECSC/DECRC.
func (g *Grid) SaveCursor()    { g.SavedRow, g.SavedCol = g.CursorRow, g.CursorCol }
func (g *Grid) RestoreCursor() { g.SetCursor(g.SavedRow, g.SavedCol) }

// EnterAltScreen/ExitAltScreen implement DECSET/DECRST 1049 (and the plainer
// 47/1047): switch the active buffer, and on entry (per the short-cut most
// terminals take for 1049) clear the alternate screen so a freshly-entered
// full-screen app doesn't see stale content from its last run.
func (g *Grid) EnterAltScreen() {
	if g.onAlt {
		return
	}
	g.onAlt = true
	g.cur = g.altBuf
	g.EraseDisplay(EraseAll)
}

func (g *Grid) ExitAltScreen() {
	if !g.onAlt {
		return
	}
	g.onAlt = false
	g.cur = g.primary
}

// OnAltScreen reports whether the alternate screen is currently active.
func (g *Grid) OnAltScreen() bool { return g.onAlt }

// Resize changes the grid's dimensions in place, truncating or padding rows
// and columns with blanks, and clamping the cursor and scroll region to fit —
// the response to a PTY TIOCSWINSZ-driven resize.
func (g *Grid) Resize(rows, cols int) {
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	g.primary = resizeRows(g.primary, rows, cols, g.blank)
	g.altBuf = resizeRows(g.altBuf, rows, cols, g.blank)
	g.Rows, g.Cols = rows, cols
	if g.onAlt {
		g.cur = g.altBuf
	} else {
		g.cur = g.primary
	}
	g.scrollTop = clamp(g.scrollTop, 0, rows-1)
	g.scrollBot = rows - 1
	g.SetCursor(g.CursorRow, g.CursorCol)
}

func resizeRows(rows [][]Cell, newRows, newCols int, blank func() Cell) [][]Cell {
	out := make([][]Cell, newRows)
	for r := 0; r < newRows; r++ {
		row := make([]Cell, newCols)
		for c := range row {
			row[c] = blank()
		}
		if r < len(rows) {
			copy(row, rows[r])
		}
		out[r] = row
	}
	return out
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
