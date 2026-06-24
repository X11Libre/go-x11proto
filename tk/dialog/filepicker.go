package dialog

import (
	"path/filepath"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
)

const dpDoubleClickMs = 400 // max gap for a double click

// FilePicker is a self-contained file-open chooser: a header showing the
// current directory, a scrollable list of entries (parent, sub-directories,
// then files) with a highlighted selection, and a key hint at the bottom. It
// draws the list itself rather than composing sub-widgets.
//
// Keys: Up/Down/PageUp/PageDown/Home/End move; Enter opens a directory or
// chooses a file; Backspace goes to the parent; Escape cancels. Mouse: a click
// selects a row, a double-click (or a click on the already-selected row)
// activates it.
//
// Fill the embedded Window (Parent/X/Y/W/H) and Font before Init. OnAccept is
// called with the chosen file path; OnCancel (optional) on Escape. The picker
// does not destroy itself — the owner closes it (Destroy) from those callbacks.
type FilePicker struct {
	tk_core.Window
	Font     *font.Font
	Keymap   *keyboard.Map
	OnAccept func(path string)
	OnCancel func()

	gc, gcHi *tk_core.GC
	dir      string
	entries  []entry
	sel      int
	top      int

	lastClickRow int
	lastClickT   base.CARD32
}

// Init creates and maps the picker and builds its GCs and keyboard map.
func (p *FilePicker) Init() error {
	p.lastClickRow = -1
	p.EventMask |= base.CARD32(event_mask.Exposure | event_mask.ButtonPress |
		event_mask.KeyPress | event_mask.StructureNotify)
	p.SetBackPixel = true
	p.BackPixel = p.Conn.X11Conn.DefaultWhitePixel()
	p.SetBorderPixel = true
	p.BorderPixel = p.Conn.X11Conn.DefaultBlackPixel()
	p.BorderWidth = 1

	p.SetWindowHandler(p)
	if err := p.Window.Create(); err != nil {
		return err
	}
	black := p.Conn.X11Conn.DefaultBlackPixel()
	white := p.Conn.X11Conn.DefaultWhitePixel()
	var err error
	if p.gc, err = p.Conn.CreateGC1(black, white, p.Font.ID); err != nil {
		return err
	}
	if p.gcHi, err = p.Conn.CreateGC1(white, black, p.Font.ID); err != nil {
		return err
	}
	if p.Keymap == nil {
		if km, err := keyboard.Load(p.Conn.X11Conn); err == nil {
			p.Keymap = km
		}
	}
	return p.Window.Map()
}

// Open shows the given directory (absolute path recommended) and takes focus.
func (p *FilePicker) Open(dir string) error {
	dir = filepath.Clean(dir)
	es, err := readDir(dir)
	if err != nil {
		return err
	}
	p.dir, p.entries, p.sel, p.top = dir, es, 0, 0
	p.Focus()
	return p.Draw()
}

// Focus requests the keyboard focus.
func (p *FilePicker) Focus() {
	_ = rpc.SetInputFocus(p.Conn.X11Conn, 2 /*RevertToParent*/, p.XID, 0)
}

// CurrentDir is the directory currently shown.
func (p *FilePicker) CurrentDir() string { return p.dir }

func (p *FilePicker) lineH() int   { return p.Font.Height() }
func (p *FilePicker) headerH() int { return p.lineH() + 4 }
func (p *FilePicker) footerH() int { return p.lineH() + 4 }

// rows is the number of entry rows that fit in the current size.
func (p *FilePicker) rows() int {
	r := (int(p.H) - p.headerH() - p.footerH()) / p.lineH()
	if r < 1 {
		return 1
	}
	return r
}

// Draw paints the header, the visible entries and the key hint.
func (p *FilePicker) Draw() error {
	if err := p.ClearArea(0, 0, 0, 0, false); err != nil {
		return err
	}
	asc := p.Font.Ascent
	// header: current directory
	p.PutText8(p.gc.XID, 4, base.INT16(asc), p.dir)
	p.FillRect(p.gc.XID, 0, base.INT16(p.headerH()-2), p.W, 1)

	// entries
	rows := p.rows()
	for r := 0; r < rows; r++ {
		i := p.top + r
		if i >= len(p.entries) {
			break
		}
		y := p.headerH() + r*p.lineH()
		gc := p.gc
		if i == p.sel {
			p.FillRect(p.gc.XID, 0, base.INT16(y), p.W, base.CARD16(p.lineH()))
			gc = p.gcHi
		}
		p.PutText8(gc.XID, 6, base.INT16(y+asc), p.entries[i].display())
	}

	// footer hint
	fy := int(p.H) - p.footerH()
	p.FillRect(p.gc.XID, 0, base.INT16(fy), p.W, 1)
	p.PutText8(p.gc.XID, 4, base.INT16(fy+2+asc), "Enter: open   Backspace: up   Esc: cancel")
	return nil
}

// HandleWindowEvent drives drawing, navigation and selection.
func (p *FilePicker) HandleWindowEvent(ev events.Event) bool {
	switch e := ev.(type) {
	case *events.ExposeEvent:
		_ = p.Draw()
	case *events.ConfigureEvent:
		p.W, p.H = e.Width, e.Height
		_ = p.Draw()
	case *events.KeyPressEvent:
		if p.Keymap != nil {
			p.handleKey(p.Keymap.Lookup(e.Key, e.State))
		}
	case *events.ButtonPressEvent:
		if e.Key == 1 {
			p.handleClick(int(e.EventY), e.Timestamp)
		}
	}
	return true
}

func (p *FilePicker) handleKey(k keyboard.Event) {
	switch k.Key {
	case keyboard.KeyUp:
		p.moveSel(-1)
	case keyboard.KeyDown:
		p.moveSel(1)
	case keyboard.KeyPageUp:
		p.moveSel(-p.rows())
	case keyboard.KeyPageDown:
		p.moveSel(p.rows())
	case keyboard.KeyHome:
		p.setSel(0)
	case keyboard.KeyEnd:
		p.setSel(len(p.entries) - 1)
	case keyboard.KeyEnter:
		p.activate(p.sel)
	case keyboard.KeyBackspace:
		p.cd(target(p.dir, ".."))
	case keyboard.KeyEscape:
		if p.OnCancel != nil {
			p.OnCancel()
		}
	default:
	}
}

func (p *FilePicker) handleClick(y int, ts base.CARD32) {
	if y < p.headerH() || y >= int(p.H)-p.footerH() {
		return
	}
	row := p.top + (y-p.headerH())/p.lineH()
	if row < 0 || row >= len(p.entries) {
		return
	}
	doubled := row == p.lastClickRow && ts-p.lastClickT <= dpDoubleClickMs
	p.lastClickRow, p.lastClickT = row, ts
	p.setSel(row)
	if doubled {
		p.activate(row)
	}
}

func (p *FilePicker) moveSel(d int) { p.setSel(p.sel + d) }

func (p *FilePicker) setSel(i int) {
	p.sel = clampSel(i, len(p.entries))
	p.top = scrollTop(p.top, p.sel, p.rows())
	_ = p.Draw()
}

// activate opens a directory entry or chooses a file entry.
func (p *FilePicker) activate(i int) {
	if i < 0 || i >= len(p.entries) {
		return
	}
	e := p.entries[i]
	if e.isDir {
		p.cd(target(p.dir, e.name))
		return
	}
	if p.OnAccept != nil {
		p.OnAccept(filepath.Join(p.dir, e.name))
	}
}

// cd navigates to dir, ignoring unreadable directories (e.g. no permission).
func (p *FilePicker) cd(dir string) {
	if es, err := readDir(dir); err == nil {
		p.dir, p.entries, p.sel, p.top = dir, es, 0, 0
		p.lastClickRow = -1
		_ = p.Draw()
	}
}
