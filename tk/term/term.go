package term

import (
	"sync"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

// Term is a terminal emulator widget: a Grid rendered onto a tk_core.Window,
// driven by a Parser fed from a PTY-spawned shell. Unlike TextView, typed
// input is never edited locally — every key is encoded (see EncodeKey) and
// written straight to the PTY; the shell/application on the other end is the
// only thing that ever changes what's on screen.
//
// Fill in Font (required) and optionally Type (defaults to XTerm256Color),
// Fg/Bg, BoldFont before Start. Start spawns Shell (defaults to $SHELL, then
// /bin/sh) and begins reading its output in a background goroutine; that
// goroutine only mutates the Grid (under Term's own lock) and signals
// Dirty() — it never touches the X connection. The caller's event loop must
// therefore select on both conn.Events() and Dirty() and call Draw() on
// either; RunLoop does exactly that as a drop-in replacement for
// conn.SimpleEventLoop() for a single-Term program.
type Term struct {
	tk_core.Window
	Font     *font.Font
	BoldFont *font.Font    // optional real bold face; nil = brighten the ANSI colour instead
	Keymap   *keyboard.Map // loaded lazily on first key press if nil
	Type     Type
	Fg, Bg   base.CARD32

	Shell    string
	ExtraEnv []string

	OnTitle func(string)
	OnExit  func(error)

	gc       *tk_core.GC
	resolver pixelResolver

	mu     sync.Mutex
	grid   *Grid
	parser *Parser
	pty    *PTY

	dirty      chan struct{}
	cols, rows int
}

// Init creates and maps the window and its GC. Call Start afterwards to
// actually spawn a shell — Init alone leaves the widget showing a blank grid,
// which is enough for size negotiation before a shell exists.
func (t *Term) Init() error {
	if t.Fg == 0 && t.Bg == 0 {
		t.Fg = t.Conn.X11Conn.DefaultBlackPixel()
		t.Bg = t.Conn.X11Conn.DefaultWhitePixel()
	}
	if t.Type.Name == "" {
		t.Type = XTerm256Color
	}
	t.dirty = make(chan struct{}, 1)
	t.resolver = newPixelResolver(t.Conn.X11Conn)

	t.EventMask |= base.CARD32(event_mask.KeyPress | event_mask.Exposure | event_mask.StructureNotify)
	t.Window.SetWindowHandler(t)
	if err := t.Window.Create(); err != nil {
		return err
	}

	gc, err := t.Conn.CreateGC1(t.Fg, t.Bg, t.Font.ID)
	if err != nil {
		return err
	}
	t.gc = gc

	t.cols, t.rows = t.cellSize()
	t.grid = NewGrid(t.rows, t.cols)
	t.grid.DefaultFg, t.grid.DefaultBg = Color{}, Color{}
	t.parser = NewParser(t.grid, t.Type)
	t.parser.Respond = func(b []byte) {
		if t.pty != nil {
			_, _ = t.pty.Master.Write(b)
		}
	}
	t.parser.SetTitle = func(s string) {
		if t.OnTitle != nil {
			t.OnTitle(s)
		}
	}
	return t.Window.Map()
}

// cellSize computes the grid dimensions the widget's current pixel size
// holds, at least 1x1.
func (t *Term) cellSize() (cols, rows int) {
	h := t.Font.Height()
	cols = max1(int(t.W) / t.Font.RuneWidth(' '))
	rows = max1(int(t.H) / h)
	return
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

// Start spawns the shell (Shell, $SHELL, or /bin/sh, in that order) on a
// fresh PTY sized to the widget's current grid, and begins the background
// read loop. Init must have been called first.
func (t *Term) Start() error {
	pty, err := OpenPTY()
	if err != nil {
		return err
	}
	if err := pty.Resize(t.rows, t.cols); err != nil {
		pty.Close()
		return err
	}
	shell := t.Shell
	if shell == "" {
		shell = shellFromEnv()
	}
	cmd, err := Spawn(pty, shell, t.ExtraEnv, t.Type.Name)
	if err != nil {
		pty.Close()
		return err
	}
	t.pty = pty
	go t.readLoop()
	go func() {
		err := cmd.Wait()
		if t.OnExit != nil {
			t.OnExit(err)
		}
	}()
	return nil
}

func (t *Term) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := t.pty.Master.Read(buf)
		if n > 0 {
			t.mu.Lock()
			t.parser.Feed(buf[:n])
			t.mu.Unlock()
			t.markDirty()
		}
		if err != nil {
			return
		}
	}
}

func (t *Term) markDirty() {
	select {
	case t.dirty <- struct{}{}:
	default: // a redraw is already pending; coalesce
	}
}

// Dirty reports when the Grid has changed and needs a Draw. The caller's
// event loop must select on it (see RunLoop).
func (t *Term) Dirty() <-chan struct{} { return t.dirty }

// RunLoop drives conn's event loop and this Term's output together: it is
// conn.SimpleEventLoop() plus reacting to Dirty(). Suitable when a program
// has exactly one Term to run for its whole lifetime (the common case: an
// lxterminal-alike top-level window; a tabbed setup runs one such process
// per tab instead of multiplexing several Terms in one connection).
func (t *Term) RunLoop(conn *core.X11Conn) {
	evCh := conn.Events()
	for {
		select {
		case ev, ok := <-evCh:
			if !ok {
				return
			}
			conn.DeliverWindowEvent(ev)
		case <-t.dirty:
			_ = t.Draw()
		}
	}
}

// Paste writes s to the PTY, wrapped in bracketed-paste markers if the
// running application has enabled DECSET 2004.
func (t *Term) Paste(s string) {
	if t.pty == nil {
		return
	}
	t.mu.Lock()
	bp := t.parser.Modes.BracketedPaste
	t.mu.Unlock()
	_, _ = t.pty.Master.Write(bracketPaste(s, bp))
}

// Draw repaints the whole grid, batching same-style runs per row so the GC's
// font/foreground only change when a cell's resolved style actually differs
// from the previous one drawn — the same technique widget.TextView's
// Highlighter path uses, applied to a 2D grid instead of per-line spans.
func (t *Term) Draw() error {
	if t.gc == nil {
		return nil
	}
	t.mu.Lock()
	rows, cols := t.grid.Rows, t.grid.Cols
	cells := make([][]Cell, rows)
	for r := range cells {
		cells[r] = append([]Cell(nil), t.grid.cur[r]...)
	}
	curRow, curCol, curVisible := t.grid.CursorRow, t.grid.CursorCol, t.grid.CursorVisible
	t.mu.Unlock()

	if err := t.ClearArea(0, 0, 0, 0, false); err != nil {
		return err
	}
	h := t.Font.Height()
	cw := t.Font.RuneWidth(' ')

	var curFont *font.Font = t.Font
	var curFg base.CARD32

	for r := 0; r < rows; r++ {
		row := cells[r]
		y := base.INT16(r * h)
		c := 0
		for c < cols {
			cell := row[c]
			start := c
			for c < cols && sameStyle(row[c], cell) {
				c++
			}
			seg := cellsText(row[start:c])
			f, fg, bg := t.styleFor(cell)
			if f != curFont {
				if err := f.SetOn(t.gc); err != nil {
					return err
				}
				curFont = f
			}
			if err := t.gc.SetBackground(bg); err != nil {
				return err
			}
			if fg != curFg {
				if err := t.gc.SetForeground(fg); err != nil {
					return err
				}
				curFg = fg
			}
			x := base.INT16(start * cw)
			// DrawTextBG already adds f.Ascent internally to convert the
			// top-left y it expects into the baseline ImageText8 needs — do
			// not add it again here (that bug drew every row one Ascent too
			// low, making the cursor block look like it sat a line above
			// the glyph it belonged to).
			if err := f.DrawTextBG(t.Drawable, t.gc.XID, x, y, seg); err != nil {
				return err
			}
			if cell.Attr&AttrUnderline != 0 {
				if err := t.FillRect(t.gc.XID, x, y+base.INT16(h-1), base.CARD16((c-start)*cw), 1); err != nil {
					return err
				}
			}
		}
	}

	if curVisible {
		if err := t.gc.SetForeground(t.resolver.Pixel(Color{}, false)); err != nil {
			return err
		}
		x, y := base.INT16(curCol*cw), base.INT16(curRow*h)
		if err := t.FillRect(t.gc.XID, x, y, base.CARD16(cw), base.CARD16(h)); err != nil {
			return err
		}
	}
	return nil
}

func sameStyle(a, b Cell) bool {
	return a.Fg == b.Fg && a.Bg == b.Bg && a.Attr == b.Attr
}

// cellsText builds the byte string to send to the core font for a run of
// cells. It must NOT UTF-8-encode: ImageText8/PolyText8 send a Go string's
// bytes verbatim as one 8-bit font character code per byte (see
// tk/core/drawable.go), so a multi-byte UTF-8 sequence would land as several
// wrong glyphs across several cells instead of one — corrupting not just
// that character but every cell after it on the line. Each rune therefore
// goes through encodeCell, which maps it onto exactly one byte.
func cellsText(cells []Cell) string {
	b := make([]byte, len(cells))
	for i, c := range cells {
		b[i] = encodeCell(c.Rune)
	}
	return string(b)
}

// encodeCell returns the single font byte for r. A core bitmap font like
// "fixed" is Latin-1: runes 0-0xFF map directly onto that byte value (the
// font's own encoding), so no translation is needed or correct there.
// Anything above Latin-1 — Unicode box-drawing/symbol runes, whether typed
// directly by a UTF-8-aware application or produced by decSpecialGraphics —
// has no glyph in such a font at all, so it degrades to a plain-ASCII
// look-alike (asciiApprox) instead of corrupting the line.
func encodeCell(r rune) byte {
	if r == 0 {
		return ' '
	}
	if r <= 0xFF {
		return byte(r)
	}
	return asciiApprox(r)
}

// asciiApprox maps a Unicode rune a Latin-1 core font cannot render to a
// plain-ASCII look-alike — crude, but legible and column-stable, which is
// what actually matters once real glyphs aren't an option.
func asciiApprox(r rune) byte {
	switch r {
	case '─', '━', '┄', '┅', '┈', '┉', '╌', '╍', '═':
		return '-'
	case '│', '┃', '┆', '┇', '┊', '┋', '╎', '╏', '║':
		return '|'
	case '┌', '┐', '└', '┘', '┼', '├', '┤', '┬', '┴',
		'┏', '┓', '┗', '┛', '╋', '┣', '┫', '┳', '┻',
		'╔', '╗', '╚', '╝', '╬', '╠', '╣', '╦', '╩',
		'╒', '╓', '╕', '╖', '╘', '╙', '╛', '╜',
		'╞', '╟', '╡', '╢', '╤', '╥', '╧', '╨', '╪', '╫':
		return '+'
	case '◆', '◇':
		return '*'
	case '▒', '░', '▓', '█':
		return '#'
	case '°':
		return 'o'
	case '±':
		return '~'
	case '·':
		return '.'
	case '≤':
		return '<'
	case '≥':
		return '>'
	case '≠':
		return '#'
	case '£':
		return 'L'
	case 'π':
		return 'p'
	}
	return '?'
}

// styleFor resolves a Cell's font and pixel colours, applying Reverse
// (swap fg/bg), Conceal (fg forced to bg, hiding the text) and Bold (either
// BoldFont if the caller supplied one, or the classic VT100 approximation of
// brightening an indexed colour into its bright variant).
func (t *Term) styleFor(cell Cell) (f *font.Font, fg, bg base.CARD32) {
	fgc, bgc := cell.Fg, cell.Bg
	if cell.Attr&AttrReverse != 0 {
		fgc, bgc = bgc, fgc
	}
	if cell.Attr&AttrBold != 0 && t.BoldFont == nil && fgc.Mode == ColorIndexed && fgc.Index < 8 {
		fgc.Index += 8
	}
	if cell.Attr&AttrConceal != 0 {
		fgc = bgc
	}
	f = t.Font
	if cell.Attr&AttrBold != 0 && t.BoldFont != nil {
		f = t.BoldFont
	}
	fg = t.resolver.Pixel(fgc, false)
	bg = t.resolver.Pixel(bgc, true)
	return
}

// HandleWindowEvent implements tk_core.WindowEventHandler.
func (t *Term) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		_ = t.Draw()
	case *events.ConfigureEvent:
		t.W, t.H = e.Width, e.Height
		t.handleResize()
	case *events.KeyPressEvent:
		t.handleKey(e)
	}
	return true
}

func (t *Term) handleResize() {
	cols, rows := t.cellSize()
	t.mu.Lock()
	changed := cols != t.cols || rows != t.rows
	if changed {
		t.cols, t.rows = cols, rows
		t.grid.Resize(rows, cols)
	}
	t.mu.Unlock()
	if changed && t.pty != nil {
		_ = t.pty.Resize(rows, cols)
	}
	_ = t.Draw()
}

func (t *Term) handleKey(e *events.KeyPressEvent) {
	if t.Keymap == nil {
		if km, err := keyboard.Load(t.Conn.X11Conn); err == nil {
			t.Keymap = km
		} else {
			return
		}
	}
	k := t.Keymap.Lookup(e.Key, e.State)
	t.mu.Lock()
	appCursor := t.parser.Modes.AppCursor
	t.mu.Unlock()
	b := EncodeKey(k, appCursor)
	if b != nil && t.pty != nil {
		_, _ = t.pty.Master.Write(b)
	}
}
