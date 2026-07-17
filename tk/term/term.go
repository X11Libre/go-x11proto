package term

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"encoding/base64"
	"strings"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"github.com/X11Libre/go-x11proto/tk/clipboard"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/font/ttf"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
)

// Term is a terminal emulator widget: a Grid rendered onto a tk_core.Window,
// driven by a Parser fed from a PTY-spawned shell. Unlike TextView, typed
// input is never edited locally — every key is encoded (see EncodeKey) and
// written straight to the PTY; the shell/application on the other end is the
// only thing that ever changes what's on screen.
//
// Fill in Font (required unless AAFace is set) and optionally Type (defaults
// to XTerm256Color), Fg/Bg, BoldFont before Start. Start spawns Shell
// (defaults to $SHELL, then /bin/sh) and begins reading its output in a
// background goroutine; that goroutine only mutates the Grid (under Term's
// own lock) and signals Dirty() — it never touches the X connection. The
// caller's event loop must therefore select on both conn.Events() and
// Dirty() and call Draw() on either; RunLoop does exactly that as a
// drop-in replacement for conn.SimpleEventLoop() for a single-Term program.
//
// Set AAFace (plus AARender, the RENDER handle AAFace composites through) to
// draw with antialiased TrueType instead of the core bitmap Font — see
// tk/font/ttf. The two rendering paths are entirely separate in Draw (drawAA
// vs the original GC-based body): they draw through different primitives
// (Picture vs GC) with little worth sharing, and keeping them apart means
// AAFace being nil is exactly the pre-existing behavior, unchanged. Font may
// be left nil when AAFace is set.
type Term struct {
	tk_core.Window
	Font     *font.Font
	BoldFont *font.Font    // optional real bold face; nil = brighten the ANSI colour instead
	Keymap   *keyboard.Map // loaded lazily on first key press if nil
	Type     Type
	Fg, Bg   base.CARD32

	AAFace       *ttf.Face
	AARender     *tk_render.Render
	FgRGB, BgRGB [3]byte

	Shell    string
	ShellArgs []string
	ExtraEnv []string

	OnTitle func(string)
	OnExit  func(error)

	// OSC 52/8/9/777 handlers, mirroring the parser callbacks. OnClipboard
	// receives the raw (base64) payload; data == "?" means the application
	// requested the current selection. OnHyperlink's uri == "" ends a link.
	OnClipboard func(selection, data string)
	OnHyperlink func(params, uri string)
	OnNotify    func(message string)
	OnOSC777    func(payload string)
	// OnMark, if set, is called with the text the user just selected with a
	// mouse drag (after the Term takes PRIMARY/CLIPBOARD ownership of it), so
	// an embedding program can log or forward it.
	OnMark func(text string)

	// oscPriPending / oscCBPending are the per-selection FIFOs of OSC 52
	// selection names (e.g. "p"/"c") whose contents an application has
	// requested via OSC 52 ; sel ; ?. As each request's SelectionNotify
	// arrives (via the matching clipboard's OnPaste in the event loop), the
	// next name is popped and the text is written back to the PTY as
	// OSC 52 ; sel ; <base64>. Guarded by oscClipMu.
	oscPriPending []string
	oscCBPending  []string
	oscClipMu     sync.Mutex

	gc       *tk_core.GC
	resolver pixelResolver
	aaPic    *tk_render.Picture
	aaFmt    tk_render.PICTFORMAT
	aaDepth  base.CARD8

	// gcBack is the offscreen pixmap the core-font (GC) draw path renders a
	// full frame into before a single CopyArea blits it onto the window —
	// the GC counterpart of aaBack. Without it every Draw would ClearArea the
	// visible window (a white flash on each redraw) and a resize/expose could
	// leave the newly revealed area blank; the backbuffer guarantees a
	// complete frame is shown in one atomic blit.
	gcBack     *tk_core.Pixmap
	gcBackDraw tk_core.Drawable
	gcBackW    base.CARD16
	gcBackH    base.CARD16

	// aaBack is an offscreen backbuffer drawAA renders into, blitted onto
	// aaPic in one shot at the end — without it, every Fill/Composite call
	// lands directly on the visible window and the partially-drawn frame
	// (background cleared, glyphs not yet drawn) is visible for a moment,
	// flickering on every redraw.
	aaBackPix *tk_core.Pixmap
	aaBackPic *tk_render.Picture
	aaBackW   base.CARD16
	aaBackH   base.CARD16

	mu     sync.Mutex
	grid   *Grid
	parser *Parser
	pty    *PTY
	cmd    *exec.Cmd // spawned shell process (set by Start)

	dirty        chan struct{}
	cols, rows   int
	scrollOffset int // lines scrolled back from the live bottom; 0 = live

	// primary is a click-drag text selection's PRIMARY-selection owner: a
	// dedicated offscreen window (same pattern demo/editor uses), so owning
	// the selection isn't tangled up with the visible window's own X
	// resource lifetime. Selection only operates on the live view
	// (scrollOffset 0) — selecting scrolled-back history is a possible
	// follow-up, not implemented here.
	clipWin base.WINDOW
	primary *clipboard.Clipboard

	// clip is the X CLIPBOARD selection (distinct from PRIMARY), used by OSC
	// 52 "c". It owns its own offscreen window so the two selections have
	// independent lifetimes.
	clipCBWin base.WINDOW
	clip      *clipboard.Clipboard

	selActive            bool
	selDragging          bool // true while the left button is held for a selection drag
	selAnchor, selCursor cellPos
}

// cellPos is a (row, col) grid position, used for the selection anchor/end.
type cellPos struct{ row, col int }

// Init creates and maps the window and its GC. Call Start afterwards to
// actually spawn a shell — Init alone leaves the widget showing a blank grid,
// which is enough for size negotiation before a shell exists.
//
// Init is idempotent for the X resource parts after a prior Detach: if the
// grid already exists (from a previous Init or a surviving Detach), only the
// X resources are recreated.
func (t *Term) Init() error {
	if t.Fg == 0 && t.Bg == 0 {
		t.Fg = t.Conn.X11Conn.DefaultBlackPixel()
		t.Bg = t.Conn.X11Conn.DefaultWhitePixel()
	}
	if t.FgRGB == ([3]byte{}) && t.BgRGB == ([3]byte{}) {
		t.BgRGB = [3]byte{0xff, 0xff, 0xff}
	}
	if t.Type.Name == "" {
		t.Type = XTerm256Color
	}
	if t.dirty == nil {
		t.dirty = make(chan struct{}, 1)
	}

	if err := t.initX(); err != nil {
		return err
	}

	if t.grid == nil {
		t.cols, t.rows = t.cellSize()
		t.grid = NewGrid(t.rows, t.cols)
		t.grid.DefaultFg, t.grid.DefaultBg = Color{}, Color{}
		if !t.Type.Scrollback {
			t.grid.scrollbackCap = 0
		}
		t.parser = NewParser(t.grid, t.Type)
		t.parser.Respond = func(b []byte) {
			if t.pty != nil {
				_, _ = t.pty.Master.Write(b)
			}
		}
		t.wireOSC()
	}
	return nil
}

// initX creates the X server resources: window, GC (if Font is set), AA
// rendering resources (if AAFace is set), and the clipboard window. It is
// called by Init and Attach and requires Drawable.Conn to be valid.
func (t *Term) initX() error {
	t.resolver = newPixelResolver(t.Conn.X11Conn)

	t.EventMask |= base.CARD32(event_mask.KeyPress | event_mask.Exposure | event_mask.StructureNotify |
		event_mask.ButtonPress | event_mask.ButtonRelease | event_mask.ButtonMotion)
	t.Window.SetWindowHandler(t)
	if err := t.Window.Create(); err != nil {
		return err
	}

	if t.Font != nil {
		gc, err := t.Conn.CreateGC1(t.Fg, t.Bg, t.Font.ID)
		if err != nil {
			return err
		}
		t.gc = gc
	}
	if t.AAFace != nil {
		if err := t.initAA(); err != nil {
			return err
		}
	}

	clipWin, err := rpc.CreateWindow1(t.Conn.X11Conn, t.Conn.X11Conn.DefaultRoot(), -10, -10, 1, 1, clipboard.EventMask)
	if err != nil {
		return err
	}
	t.clipWin = clipWin
	primary, err := clipboard.New(t.Conn.X11Conn, clipWin, "PRIMARY")
	if err != nil {
		return err
	}
	t.primary = primary
	primary.OnPaste = func(s string) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("term: recovered from PRIMARY OnPaste panic: %v", r)
			}
		}()
		// An OSC 52 ; sel ; ? query in flight takes priority: its answer is
		// the selection text written back to the PTY, not pasted as keystrokes.
		t.oscClipMu.Lock()
		if n := len(t.oscPriPending); n > 0 {
			sel := t.oscPriPending[0]
			t.oscPriPending = t.oscPriPending[1:]
			t.oscClipMu.Unlock()
			t.writeOSC52(sel, s)
			return
		}
		t.oscClipMu.Unlock()
		t.Paste(s)
	}

	clipCBWin, err := rpc.CreateWindow1(t.Conn.X11Conn, t.Conn.X11Conn.DefaultRoot(), -10, -10, 1, 1, clipboard.EventMask)
	if err != nil {
		return err
	}
	t.clipCBWin = clipCBWin
	clip, err := clipboard.New(t.Conn.X11Conn, clipCBWin, "CLIPBOARD")
	if err != nil {
		return err
	}
	t.clip = clip
	clip.OnPaste = func(s string) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("term: recovered from CLIPBOARD OnPaste panic: %v", r)
			}
		}()
		// Only OSC 52 CLIPBOARD queries are answered here; the terminal has
		// no other use for inbound CLIPBOARD pastes.
		t.oscClipMu.Lock()
		if n := len(t.oscCBPending); n > 0 {
			sel := t.oscCBPending[0]
			t.oscCBPending = t.oscCBPending[1:]
			t.oscClipMu.Unlock()
			t.writeOSC52(sel, s)
			return
		}
		t.oscClipMu.Unlock()
	}
	t.Conn.X11Conn.RegisterWindowHandler(clipCBWin, clip)
	t.Conn.X11Conn.RegisterWindowHandler(clipWin, primary)

	t.cols, t.rows = t.cellSize()
	if t.grid != nil {
		t.grid.Resize(t.rows, t.cols)
	}
	return t.Window.Map()
}

// initAA creates the AA rendering resources (window picture, backbuffer).
func (t *Term) initAA() error {
	geom, err := t.Window.GetGeometry()
	if err != nil {
		return err
	}
	fmtID, err := t.AARender.StandardFormat(geom.Depth, false)
	if err != nil {
		return err
	}
	pic, err := t.AARender.PictureFor(t.Window.Drawable, fmtID, tk_render.PictureValues{})
	if err != nil {
		return err
	}
	t.aaPic = pic
	t.aaFmt = fmtID
	t.aaDepth = geom.Depth
	return nil
}

// Detach destroys all X server resources (window, GC, AA pictures, clipboard
// window) held by this Term. The shell, PTY, grid, and parser continue
// running — the grid keeps updating as the shell produces output, but no
// window is visible. Call Attach to create a new window, possibly on a
// different X display.
//
// Detach must be called while the X11 connection is still alive (all
// Destroy/Free requests need a working connection). The caller may close the
// connection afterwards.
func (t *Term) Detach() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.primary = nil
	t.clip = nil

	if t.aaBackPic != nil {
		_ = t.aaBackPic.Free()
		t.aaBackPic = nil
	}
	if t.aaBackPix != nil {
		_ = t.aaBackPix.Free()
		t.aaBackPix = nil
	}
	if t.aaPic != nil {
		_ = t.aaPic.Free()
		t.aaPic = nil
	}
	if t.gc != nil {
		_ = t.gc.Free()
		t.gc = nil
	}
	if t.gcBack != nil {
		_ = t.gcBack.Free()
		t.gcBack = nil
		t.gcBackDraw = tk_core.Drawable{}
	}
	if t.clipCBWin != 0 {
		_ = rpc.DestroyWindow(t.Conn.X11Conn, t.clipCBWin)
		t.clipCBWin = 0
	}
	if t.clipWin != 0 {
		_ = rpc.DestroyWindow(t.Conn.X11Conn, t.clipWin)
		t.clipWin = 0
	}
	if t.XID != 0 {
		_ = t.Window.Destroy()
		t.XID = 0
		t.Drawable.XID = 0
		t.Drawable.Conn = nil
	}
	return nil
}

// ResetForReattach clears the Term's cached X resources (AA/RENDER pictures,
// GC, window XID, drawable connection) WITHOUT issuing any X request. It is
// used after the X connection died on its own (e.g. the user closed the
// window) so a subsequent Attach rebuilds every resource fresh on a new
// connection. Calling t.Detach() in that situation would panic, because the
// socket is already gone and the Destroy/Free calls would dereference nil.
func (t *Term) ResetForReattach() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.primary = nil
	t.clip = nil
	t.aaBackPic = nil
	t.aaBackPix = nil
	t.aaPic = nil
	t.aaFmt = 0
	t.aaDepth = 0
	t.aaBackW = 0
	t.aaBackH = 0
	t.gc = nil
	t.gcBack = nil
	t.gcBackDraw = tk_core.Drawable{}
	t.clipCBWin = 0
	t.clipWin = 0
	t.XID = 0
	t.Drawable.XID = 0
	t.Drawable.Conn = nil
}

// Attach creates new X server resources for this Term on tk's X11 connection,
// which must be a brand-new (or at least not the previously-detached)
// connection. The caller must set Font, AAFace, AARender, Fg, Bg etc. on t
// before calling Attach, just as for Init; Grid and Parser state is preserved
// from before Detach.
//
// parent is the XID of the parent window (typically the root window of tk).
// After Attach the window is mapped and the current grid is redrawn.
func (t *Term) Attach(tk *tk_core.TkConn, parent base.WINDOW) error {
	t.mu.Lock()

	t.Drawable.Conn = tk
	t.ParentXID = parent
	t.Parent = nil

	if err := t.initX(); err != nil {
		t.mu.Unlock()
		return err
	}
	// Release the lock before Draw: drawAA takes t.mu again to read the grid,
	// so holding it here would deadlock (Attach -> initX -> Draw -> drawAA).
	t.mu.Unlock()
	_ = t.Draw()
	return nil
}

// cellSize computes the grid dimensions the widget's current pixel size
// holds, at least 1x1. Returns a default 80x24 when neither Font nor AAFace
// is set yet (e.g. before Init/Attach).
func (t *Term) cellSize() (cols, rows int) {
	var h, cw int
	if t.AAFace != nil {
		h, cw = t.AAFace.Height(), t.AAFace.Advance(' ')
	} else if t.Font != nil {
		h, cw = t.Font.Height(), t.Font.RuneWidth(' ')
	} else {
		return 80, 24
	}
	cols = max1(int(t.W) / cw)
	rows = max1(int(t.H) / h)
	return
}

// InitTerm initialises the terminal's grid and parser without creating any
// X server resources. Call before Start when no X11 connection is available
// yet (detached mode). After a connection is established, call Init or
// Attach to create the X window and rendering resources.
func (t *Term) InitTerm() error {
	if t.Type.Name == "" {
		t.Type = XTerm256Color
	}
	if t.dirty == nil {
		t.dirty = make(chan struct{}, 1)
	}
	if t.grid != nil {
		return nil
	}
	t.cols, t.rows = t.cellSize()
	t.grid = NewGrid(t.rows, t.cols)
	t.grid.DefaultFg, t.grid.DefaultBg = Color{}, Color{}
	if !t.Type.Scrollback {
		t.grid.scrollbackCap = 0
	}
	t.parser = NewParser(t.grid, t.Type)
	t.parser.Respond = func(b []byte) {
		if t.pty != nil {
			_, _ = t.pty.Master.Write(b)
		}
	}
	t.wireOSC()
	return nil
}

// wireOSC hooks the parser's OSC callbacks up to the Term's On* handlers and,
// for OSC 52, to the X11 PRIMARY selection. Shared by Init and Attach so both
// code paths configure the same behaviour.
func (t *Term) wireOSC() {
	t.parser.SetTitle = func(s string) {
		if t.OnTitle != nil {
			t.OnTitle(s)
		}
	}
	t.parser.SetClipboard = func(sel, data string) {
		if t.OnClipboard != nil {
			t.OnClipboard(sel, data)
		}
		if data == "?" || t.primary == nil || t.clip == nil {
			return
		}
		if dec, err := base64.StdEncoding.DecodeString(data); err == nil {
			t.ownSelection(sel, string(dec))
		}
	}
	t.parser.RequestClipboard = func(sel string) {
		if t.OnClipboard != nil {
			t.OnClipboard(sel, "?")
		}
		if t.primary == nil || t.clip == nil {
			return
		}
		// Route the query to the right X selection(s) by the Pc selector:
		// 'p' -> PRIMARY, 'c' -> CLIPBOARD, 's' -> both, bare/numeric -> CLIPBOARD.
		wantPrimary := strings.ContainsRune(sel, 'p') || strings.ContainsRune(sel, 'P')
		wantClip := strings.ContainsRune(sel, 'c') || strings.ContainsRune(sel, 'C')
		if strings.ContainsRune(sel, 's') || strings.ContainsRune(sel, 'S') {
			wantPrimary, wantClip = true, true
		}
		if !wantPrimary && !wantClip {
			wantClip = true // default a bare/numeric query to CLIPBOARD
		}
		if wantPrimary {
			t.requestOSCClip(sel, t.primary, &t.oscPriPending)
		}
		if wantClip {
			t.requestOSCClip(sel, t.clip, &t.oscCBPending)
		}
	}
	t.parser.SetHyperlink = func(params, uri string) {
		if t.OnHyperlink != nil {
			t.OnHyperlink(params, uri)
		}
	}
	t.parser.Notify = func(msg string) {
		if t.OnNotify != nil {
			t.OnNotify(msg)
		}
	}
	t.parser.OSC777 = func(payload string) {
		if t.OnOSC777 != nil {
			t.OnOSC777(payload)
		}
	}
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
	cmd, err := Spawn(pty, shell, t.ShellArgs, t.ExtraEnv, t.Type.Name)
	if err != nil {
		pty.Close()
		return err
	}
	t.pty = pty
	t.cmd = cmd
	go t.readLoop()
	go func() {
		err := cmd.Wait()
		if t.OnExit != nil {
			t.OnExit(err)
		}
	}()
	return nil
}

// Stop terminates the spawned shell: it closes the PTY master (the shell sees
// EOF and exits) and signals the whole process group (the shell is spawned as
// a session leader via Setsid, so a single-process signal is often ignored).
// It sends SIGTERM first and, if the process is still alive after a short
// grace period, escalates to SIGKILL. Safe to call multiple times; a no-op if
// the shell was never started.
func (t *Term) Stop() error {
	t.mu.Lock()
	cmd := t.cmd
	pty := t.pty
	t.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		pid := cmd.Process.Pid
		// Negative pid targets the whole process group created by Setsid.
		_ = syscall.Kill(-pid, syscall.SIGTERM)
		_ = cmd.Process.Signal(syscall.SIGTERM)
		// Give the shell a brief moment to shut down gracefully, then force
		// it. A goroutine avoids blocking Stop on the shell's exit.
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			_ = cmd.Process.Signal(syscall.SIGKILL)
		}()
	}
	if pty != nil {
		_ = pty.Close()
	}
	return nil
}

func (t *Term) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := t.pty.Master.Read(buf)
		if n > 0 {
			t.mu.Lock()
			if t.parser != nil {
				t.parser.Feed(buf[:n])
			}
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

// X11 delivers wheel/touchpad scrolling as button events: buttons 4/5 are
// vertical up/down (see tk/widget/scroll.go for the same convention).
const (
	btnWheelUp   base.CARD8 = 4
	btnWheelDown base.CARD8 = 5
)

// wheelStepLines is how many lines one wheel/touchpad notch scrolls.
const wheelStepLines = 3

// ScrollTo scrolls to offset lines back from the live bottom (0, clamped),
// redrawing if the (clamped) offset actually changed.
func (t *Term) ScrollTo(offset int) {
	if offset < 0 {
		offset = 0
	}
	t.mu.Lock()
	if max := t.grid.ScrollbackLen(); offset > max {
		offset = max
	}
	changed := offset != t.scrollOffset
	t.scrollOffset = offset
	t.mu.Unlock()
	if changed {
		_ = t.Draw()
	}
}

// ScrollBy scrolls by delta lines: positive scrolls back into history,
// negative scrolls forward toward the live bottom.
func (t *Term) ScrollBy(delta int) {
	t.mu.Lock()
	cur := t.scrollOffset
	t.mu.Unlock()
	t.ScrollTo(cur + delta)
}

// SetFontSize rescales the antialiased font (zoom) to px points, keeping the
// window size fixed so more or fewer cells fit — like xterm's Ctrl +/-
// zoom. The grid, PTY, and next draw are updated to the new cell metrics.
// It is a no-op (returning an error) when the terminal is using the core X
// font, which is not freely scalable.
func (t *Term) SetFontSize(px float64) error {
	if t.AAFace == nil {
		return fmt.Errorf("term: SetFontSize requires an AA face")
	}
	if px < 6 {
		px = 6
	}
	if px > 72 {
		px = 72
	}
	if err := t.AAFace.Resize(px); err != nil {
		return err
	}
	t.mu.Lock()
	cols, rows := t.cellSize()
	changed := cols != t.cols || rows != t.rows
	if changed {
		t.cols, t.rows = cols, rows
		t.grid.Resize(rows, cols)
	}
	t.mu.Unlock()
	if changed && t.pty != nil {
		_ = t.pty.Resize(rows, cols)
	}
	return t.Draw()
}

// visibleRows returns exactly t.rows row-slices to render: live unmodified
// when scrollOffset is 0 (the common case), otherwise a window blending
// scrollback with the top of live, oldest at the top — mirrors
// widget.TextView's ScrollTo model, except offset 0 means "at the live
// bottom" instead of "at the top" (a terminal's natural resting position).
//
// The offset is relative to the current live bottom, not a fixed point in
// history: new output arriving while scrolled back shifts the view along
// with it, rather than holding on a fixed absolute line. Real terminals
// usually do the latter; this is simpler and still lets you read back
// through history, which is the actual ask — pinning to an absolute line
// across new output is a possible follow-up, not done here.
func (t *Term) visibleRows(live [][]Cell) [][]Cell {
	k := t.scrollOffset
	if k == 0 {
		return live
	}
	sb := t.grid.ScrollbackLines(k)
	if k >= t.rows {
		return sb[:t.rows]
	}
	window := make([][]Cell, 0, t.rows)
	window = append(window, sb...)
	window = append(window, live[:t.rows-k]...)
	return window
}

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
			// A panic in event handling (e.g. a selection RPC or a bad
			// event) must never take down the whole loop — that would stop
			// every redraw. Recover, log, and keep looping.
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("term: recovered from event handler panic: %v", r)
					}
				}()
				conn.DeliverWindowEvent(ev)
			}()
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

// writeOSC52 answers an OSC 52 selection query by writing
// "OSC 52 ; sel ; <base64> ST" back to the PTY, so the application receives
// the current selection contents (or an empty one if none).
func (t *Term) writeOSC52(sel, data string) {
	if t.pty == nil {
		return
	}
	resp := "\x1b]52;" + sel + ";" + base64.StdEncoding.EncodeToString([]byte(data)) + "\x07"
	_, _ = t.pty.Master.Write([]byte(resp))
}

// ownSelection takes ownership of the appropriate X selection(s) for an OSC 52
// set, based on the Pc selector: 'p' -> PRIMARY, 'c' -> CLIPBOARD, 's' -> both,
// a bare/numeric selector -> CLIPBOARD. As a convenience for middle-click paste
// (which reads PRIMARY), an explicit 'c' also populates PRIMARY.
func (t *Term) ownSelection(sel, text string) {
	wantPrimary := strings.ContainsRune(sel, 'p') || strings.ContainsRune(sel, 'P')
	wantClip := strings.ContainsRune(sel, 'c') || strings.ContainsRune(sel, 'C')
	if strings.ContainsRune(sel, 's') || strings.ContainsRune(sel, 'S') {
		wantPrimary, wantClip = true, true
	}
	if wantClip && !wantPrimary {
		wantPrimary = true // "c" also feeds PRIMARY so middle-click paste works
	}
	if !wantPrimary && !wantClip {
		wantClip = true // bare numeric selectors target CLIPBOARD
	}
	if wantPrimary && t.primary != nil {
		go func() { _ = t.primary.Own(text) }()
	}
	if wantClip && t.clip != nil {
		go func() { _ = t.clip.Own(text) }()
	}
}

// requestOSCClip issues an async OSC 52 selection query against cb, queueing
// sel so the matching clipboard's OnPaste knows which OSC 52 response to write.
// If there is no owner (or the request fails) it responds immediately with an
// empty value and drops the queued entry.
func (t *Term) requestOSCClip(sel string, cb *clipboard.Clipboard, q *[]string) {
	t.oscClipMu.Lock()
	*q = append(*q, sel)
	t.oscClipMu.Unlock()
	ok, err := cb.RequestText()
	if err != nil {
		t.popPending(q)
		return
	}
	if !ok {
		t.popPending(q)
		t.writeOSC52(sel, "")
	}
}

// popPending removes the oldest queued OSC 52 query name from q.
func (t *Term) popPending(q *[]string) {
	t.oscClipMu.Lock()
	if len(*q) > 0 {
		*q = (*q)[1:]
	}
	t.oscClipMu.Unlock()
}

// Draw repaints the whole grid, batching same-style runs per row so the GC's
// font/foreground only change when a cell's resolved style actually differs
// from the previous one drawn — the same technique widget.TextView's
// Highlighter path uses, applied to a 2D grid instead of per-line spans.
func (t *Term) Draw() error {
	if t.AAFace != nil {
		return t.drawAA()
	}
	if t.gc == nil {
		return nil
	}
	if err := t.ensureGCBackBuffer(); err != nil {
		return err
	}
	back := t.gcBackDraw

	t.mu.Lock()
	rows, cols := t.grid.Rows, t.grid.Cols
	live := make([][]Cell, rows)
	for r := range live {
		live[r] = append([]Cell(nil), t.grid.cur[r]...)
	}
	curRow, curCol := t.grid.CursorRow, t.grid.CursorCol
	// The cursor sits at a live grid position, meaningless once a scrolled-
	// back view is showing history instead — hide it, like real terminals do.
	curVisible := t.grid.CursorVisible && t.scrollOffset == 0
	cells := t.visibleRows(live)
	t.mu.Unlock()

	// Clear the whole backbuffer to the default background (isBg selects the
	// same monochrome fallback the default cell background resolves to, so the
	// uncleared remainder — the strip right of the last column and below the
	// last row — matches the content instead of showing through black).
	if err := t.gc.SetForeground(t.resolver.Pixel(Color{}, true)); err != nil {
		return err
	}
	if err := back.FillRect(t.gc.XID, 0, 0, base.CARD16(t.W), base.CARD16(t.H)); err != nil {
		return err
	}
	h := t.Font.Height()
	cw := t.Font.RuneWidth(' ')

	var curFont *font.Font = t.Font
	var curFg base.CARD32 = ^base.CARD32(0) // sentinel: force first SetForeground
	var curBg base.CARD32 = ^base.CARD32(0)

	for r := 0; r < rows; r++ {
		row := cells[r]
		y := base.INT16(r * h)
		if !t.selActive {
			// Fast path: batch same-style runs.
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
				if bg != curBg {
					if err := t.gc.SetBackground(bg); err != nil {
						return err
					}
					curBg = bg
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
				if err := f.DrawTextBG(back, t.gc.XID, x, y, seg); err != nil {
					return err
				}
				if cell.Attr&AttrUnderline != 0 {
					if err := back.FillRect(t.gc.XID, x, y+base.INT16(h-1), base.CARD16((c-start)*cw), 1); err != nil {
						return err
					}
				}
			}
			continue
		}
		// Selection active: draw per-cell so reverse-video follows the exact
		// selected range. Run batching would over-/under-highlight at the
		// selection edges and mis-invert colored runs (e.g. a whole colored
		// prompt path instead of just the dragged part).
		for c := 0; c < cols; c++ {
			cell := row[c]
			f, fg, bg := t.styleFor(cell)
			if t.selectedAt(r, c) {
				fg, bg = bg, fg
			}
			if f != curFont {
				if err := f.SetOn(t.gc); err != nil {
					return err
				}
				curFont = f
			}
			if bg != curBg {
				if err := t.gc.SetBackground(bg); err != nil {
					return err
				}
				curBg = bg
			}
			if fg != curFg {
				if err := t.gc.SetForeground(fg); err != nil {
					return err
				}
				curFg = fg
			}
			x := base.INT16(c * cw)
			if err := f.DrawTextBG(back, t.gc.XID, x, y, cellsText([]Cell{cell})); err != nil {
				return err
			}
			if cell.Attr&AttrUnderline != 0 {
				if err := back.FillRect(t.gc.XID, x, y+base.INT16(h-1), base.CARD16(cw), 1); err != nil {
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
		if err := back.FillRect(t.gc.XID, x, y, base.CARD16(cw), base.CARD16(h)); err != nil {
			return err
		}
	}

	// Blit the finished frame onto the visible window in a single CopyArea.
	return back.CopyArea(t.Drawable.XID, t.gc.XID, 0, 0, 0, 0, base.CARD16(t.W), base.CARD16(t.H))
}

func sameStyle(a, b Cell) bool {
	return a.Fg == b.Fg && a.Bg == b.Bg && a.Attr == b.Attr
}

// drawAA is Draw's antialiased-TrueType counterpart, used instead of the
// GC-based body above when AAFace is set. It composites through aaBackPic (a
// RENDER Picture over an offscreen pixmap) rather than a GC, so it is a
// separate implementation rather than a shared code path with branches — the
// two draw through different primitives with little worth factoring out. The
// finished frame is blitted onto the real window (aaPic) in one Composite at
// the end — see ensureAABackBuffer's doc comment for why.
//
// Unlike the core path, cell text here is real UTF-8 (runesText), not
// squeezed through encodeCell's Latin-1 degradation: an antialiased font can
// actually have glyphs beyond Latin-1, so there's no need to approximate.
func (t *Term) drawAA() error {
	if t.aaPic == nil {
		return nil
	}
	if err := t.ensureAABackBuffer(); err != nil {
		return err
	}
	back := t.aaBackPic

	t.mu.Lock()
	rows, cols := t.grid.Rows, t.grid.Cols
	live := make([][]Cell, rows)
	for r := range live {
		live[r] = append([]Cell(nil), t.grid.cur[r]...)
	}
	curRow, curCol := t.grid.CursorRow, t.grid.CursorCol
	curVisible := t.grid.CursorVisible && t.scrollOffset == 0
	cells := t.visibleRows(live)
	t.mu.Unlock()

	h := t.AAFace.Height()
	cw := t.AAFace.Advance(' ')
	ascent := t.AAFace.Ascent()

	if err := back.Fill(tk_render.OpSrc, rgbColor(t.BgRGB),
		[]base.Rectangle{{X: 0, Y: 0, Width: t.aaBackW, Height: t.aaBackH}}); err != nil {
		return err
	}

	for r := 0; r < rows; r++ {
		row := cells[r]
		y := r * h
		if !t.selActive {
			// Fast path: batch same-style runs.
			c := 0
			for c < cols {
				cell := row[c]
				start := c
				for c < cols && sameStyle(row[c], cell) {
					c++
				}
				fg, bg := t.styleForAA(cell)
				x := start * cw
				width := (c - start) * cw
				if bg != t.BgRGB {
					if err := back.Fill(tk_render.OpOver, rgbColor(bg),
						[]base.Rectangle{{X: base.INT16(x), Y: base.INT16(y), Width: base.CARD16(width), Height: base.CARD16(h)}}); err != nil {
						return err
					}
				}
				if _, err := t.AAFace.DrawString(back, x, y+ascent, runesText(row[start:c]), fg); err != nil {
					return err
				}
				if cell.Attr&AttrUnderline != 0 {
					if err := back.Fill(tk_render.OpOver, rgbColor(fg),
						[]base.Rectangle{{X: base.INT16(x), Y: base.INT16(y + h - 1), Width: base.CARD16(width), Height: 1}}); err != nil {
						return err
					}
				}
			}
			continue
		}
		// Selection active: draw per-cell so reverse-video follows the exact
		// selected range (run batching would mis-highlight colored runs).
		for c := 0; c < cols; c++ {
			cell := row[c]
			fg, bg := t.styleForAA(cell)
			if t.selectedAt(r, c) {
				fg, bg = bg, fg
			}
			x := c * cw
			if bg != t.BgRGB {
				if err := back.Fill(tk_render.OpOver, rgbColor(bg),
					[]base.Rectangle{{X: base.INT16(x), Y: base.INT16(y), Width: base.CARD16(cw), Height: base.CARD16(h)}}); err != nil {
					return err
				}
			}
			if _, err := t.AAFace.DrawString(back, x, y+ascent, runesText([]Cell{cell}), fg); err != nil {
				return err
			}
			if cell.Attr&AttrUnderline != 0 {
				if err := back.Fill(tk_render.OpOver, rgbColor(fg),
					[]base.Rectangle{{X: base.INT16(x), Y: base.INT16(y + h - 1), Width: base.CARD16(cw), Height: 1}}); err != nil {
					return err
				}
			}
		}
	}

	if curVisible {
		x, y := curCol*cw, curRow*h
		if err := back.Fill(tk_render.OpOver, rgbColor(t.FgRGB),
			[]base.Rectangle{{X: base.INT16(x), Y: base.INT16(y), Width: base.CARD16(cw), Height: base.CARD16(h)}}); err != nil {
			return err
		}
	}

	return t.aaPic.Composite(tk_render.OpSrc, back, nil, 0, 0, 0, 0, 0, 0, t.aaBackW, t.aaBackH)
}

// ensureAABackBuffer (re)creates the offscreen pixmap+picture drawAA renders
// into when the window's size has changed (or on first use), so drawAA
// always has a same-size backbuffer to draw a full frame into before a
// single blit makes it visible — without this, every individual Fill/
// Composite call in drawAA would land directly on the visible window, and
// the half-drawn frame (background cleared, glyphs not yet drawn) would be
// visible for a moment: visible flicker on every redraw, worst on every
// keypress since each one triggers a full repaint.
func (t *Term) ensureAABackBuffer() error {
	w, h := t.W, t.H
	if t.aaBackPic != nil && t.aaBackW == w && t.aaBackH == h {
		return nil
	}
	if t.aaBackPic != nil {
		_ = t.aaBackPic.Free()
		_ = t.aaBackPix.Free()
	}
	pix, err := t.Conn.CreatePixmap(t.aaDepth, base.DRAWABLE(t.Conn.X11Conn.DefaultRoot()), w, h)
	if err != nil {
		return err
	}
	pic, err := t.AARender.PictureFor(pix.Drawable, t.aaFmt, tk_render.PictureValues{})
	if err != nil {
		_ = pix.Free()
		return err
	}
	t.aaBackPix, t.aaBackPic, t.aaBackW, t.aaBackH = pix, pic, w, h
	return nil
}

// ensureGCBackBuffer (re)creates the offscreen pixmap the core-font (GC) draw
// path renders a full frame into when the window's size has changed (or on
// first use), so Draw always has a same-size backbuffer to draw into before a
// single CopyArea makes it visible — without this, every individual Fill/
// ImageText8 in Draw would land directly on the visible window, and the
// half-drawn frame (background cleared, glyphs not yet drawn) would be visible
// for a moment: flicker on every redraw, worst on every keypress since each
// one triggers a full repaint. It mirrors ensureAABackBuffer.
func (t *Term) ensureGCBackBuffer() error {
	w, h := t.W, t.H
	if t.gcBack != nil && t.gcBackW == base.CARD16(w) && t.gcBackH == base.CARD16(h) {
		return nil
	}
	if t.gcBack != nil {
		_ = t.gcBack.Free()
	}
	geom, err := t.Window.GetGeometry()
	if err != nil {
		return err
	}
	pix, err := t.Conn.CreatePixmap(geom.Depth, base.DRAWABLE(t.Conn.X11Conn.DefaultRoot()), w, h)
	if err != nil {
		return err
	}
	t.gcBack, t.gcBackDraw, t.gcBackW, t.gcBackH = pix, tk_core.Drawable{Conn: t.Conn, XID: pix.Drawable.XID}, base.CARD16(w), base.CARD16(h)
	return nil
}

// styleForAA is styleFor's RGB counterpart for the AA path: same
// reverse/bold/conceal resolution, but ending in resolveRGB instead of a
// pixelResolver (RENDER composites real RGB, not a visual-dependent pixel
// value, so there's no need for pixelResolver's TrueColor/DirectColor mask
// math here at all).
func (t *Term) styleForAA(cell Cell) (fg, bg [3]byte) {
	fgc, bgc := cell.Fg, cell.Bg
	if cell.Attr&AttrReverse != 0 {
		fgc, bgc = bgc, fgc
	}
	if cell.Attr&AttrBold != 0 && fgc.Mode == ColorIndexed && fgc.Index < 8 {
		fgc.Index += 8
	}
	if cell.Attr&AttrConceal != 0 {
		fgc = bgc
	}
	return t.resolveRGB(fgc, false), t.resolveRGB(bgc, true)
}

// resolveRGB is pixelResolver.Pixel's RGB counterpart.
func (t *Term) resolveRGB(c Color, isBg bool) [3]byte {
	switch c.Mode {
	case ColorRGB:
		return [3]byte{c.R, c.G, c.B}
	case ColorIndexed:
		r, g, b := indexedRGB(c.Index)
		return [3]byte{r, g, b}
	default: // ColorDefault
		if isBg {
			return t.BgRGB
		}
		return t.FgRGB
	}
}

// rgbColor converts an opaque RGB triplet to a RENDER Color.
func rgbColor(c [3]byte) tk_render.Color {
	return tk_render.Color{
		Red: base.CARD16(c[0]) * 0x101, Green: base.CARD16(c[1]) * 0x101, Blue: base.CARD16(c[2]) * 0x101,
		Alpha: 0xffff,
	}
}

// runesText builds a plain UTF-8 string for a run of cells, unlike cellsText
// which encodes for the Latin-1 core font — the AA path draws real Unicode.
func runesText(cells []Cell) string {
	rs := make([]rune, len(cells))
	for i, c := range cells {
		if c.Rune == 0 {
			rs[i] = ' '
		} else {
			rs[i] = c.Rune
		}
	}
	return string(rs)
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
	case *events.ButtonPressEvent:
		t.handleButtonPress(e)
	case *events.ButtonReleaseEvent:
		t.handleButtonRelease(e)
	case *events.MotionEvent:
		t.handleMotion(e)
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
	// Ctrl + '+'/'= zooms in, Ctrl + '-' zooms out (font rescale; window
	// stays fixed). Intercept before the key reaches the PTY.
	if k.Ctrl && t.AAFace != nil {
		switch k.Rune {
		case '+', '=':
			_ = t.SetFontSize(t.AAFace.Size() + 2)
			return
		case '-':
			_ = t.SetFontSize(t.AAFace.Size() - 2)
			return
		}
	}
	t.mu.Lock()
	appCursor := t.parser.Modes.AppCursor
	t.mu.Unlock()
	b := EncodeKey(k, appCursor)
	if b != nil && t.pty != nil {
		_, _ = t.pty.Master.Write(b)
		// Typing while scrolled back into history snaps back to live output,
		// same as real terminals — otherwise the input you just sent would
		// happen off-screen, above the current (stale) view.
		t.ScrollTo(0)
	}
}
