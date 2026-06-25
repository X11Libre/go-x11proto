package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"github.com/X11Libre/go-x11proto/tk/clipboard"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/dialog"
	"github.com/X11Libre/go-x11proto/tk/font"
	"github.com/X11Libre/go-x11proto/tk/theme"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

const (
	barH = 22 // menu bar / status line height
	sbW  = 16 // scrollbar width
	winX = 100
	winY = 100
	winW = 700
	winH = 500
)

// Editor wires the toolkit widgets into a working text editor.
type Editor struct {
	conn *core.X11Conn
	tk   *tk_core.TkConn

	frame  *tk_widget.Frame
	menu   *tk_widget.MenuBar
	tv     *tk_widget.TextView
	sb     *tk_widget.Scrollbar
	status *tk_widget.Label

	font     *font.Font
	statusGc *tk_core.GC

	clip    *clipboard.Clipboard
	clipWin base.WINDOW

	prompt  *promptBox         // active modal filename prompt, nil when none
	picker  *dialog.FilePicker // active open dialog, nil when none
	confirm *dialog.Confirm    // active confirm dialog, nil when none

	filename  string
	modified  bool
	lastQuery string // last Find / Replace search term
}

// Init builds the window, widgets and wiring, then loads filename (if any).
func (e *Editor) Init(filename string) error {
	e.filename = filename

	// Font: a monospace face at the desktop's themed size, "fixed" as fallback.
	th := theme.Load(e.conn)
	e.font = openMono(e.conn, th.FontPixelSize())

	// Top-level frame (border layout). Parent nil -> root.
	e.frame = &tk_widget.Frame{Window: tk_core.Window{
		Drawable: tk_core.Drawable{Conn: e.tk},
		Name:     "go-xedit",
		X:        winX, Y: winY, W: winW, H: winH,
	}}
	if err := e.frame.Init(); err != nil {
		return err
	}
	_ = e.frame.SetIcon(editorIcon())

	if err := e.buildMenu(); err != nil {
		return err
	}
	if err := e.buildText(); err != nil {
		return err
	}
	if err := e.buildScrollbar(); err != nil {
		return err
	}
	if err := e.buildStatus(); err != nil {
		return err
	}

	// Border layout: menu on top, status at the bottom, scrollbar on the right,
	// text filling the middle.
	e.frame.Top = &tk_widget.Slot{Win: &e.menu.Window, Extent: barH}
	e.frame.Bottom = &tk_widget.Slot{Win: &e.status.Window, Extent: barH}
	e.frame.Right = &tk_widget.Slot{Win: &e.sb.Window, Extent: sbW}
	e.frame.Center = &e.tv.Window
	e.frame.Relayout(int(e.frame.W), int(e.frame.H))

	e.wireCallbacks()
	if err := e.setupClipboard(); err != nil {
		return err
	}

	if filename != "" {
		e.loadFile(filename)
	} else {
		e.tv.SetText("")
	}
	e.refresh()
	return nil
}

func (e *Editor) buildMenu() error {
	e.menu = &tk_widget.MenuBar{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: e.tk}, Parent: &e.frame.Window, X: 0, Y: 0, W: winW,
		},
		TearOff: true, // all menus detachable (bar-wide; or wire to theme.TearOffMenus)
	}
	e.menu.AddMenu("File", []tk_widget.MenuItem{
		{Label: "New", Accel: "Ctrl+N", OnClick: e.newFile},
		{Label: "Open", Accel: "Ctrl+O", OnClick: e.open},
		{Label: "Save", Accel: "Ctrl+S", OnClick: e.save},
		{Label: "Save As", Accel: "Ctrl+Shift+S", OnClick: e.saveAs},
		{Separator: true},
		{Label: "Quit", Accel: "Ctrl+Q", OnClick: e.quit},
	})
	e.menu.AddMenu("Edit", []tk_widget.MenuItem{
		{Label: "Undo", Accel: "Ctrl+Z", OnClick: func() { e.tv.Undo() }},
		{Label: "Redo", Accel: "Ctrl+Y", OnClick: func() { e.tv.Redo() }},
		{Separator: true},
		{Label: "Cut", Accel: "Ctrl+X", OnClick: e.cut},
		{Label: "Copy", Accel: "Ctrl+C", OnClick: e.copy},
		{Label: "Paste", Accel: "Ctrl+V", OnClick: e.paste},
		{Separator: true},
		{Label: "Select All", Accel: "Ctrl+A", OnClick: func() { e.tv.SelectAll() }},
		{Label: "Find", Accel: "Ctrl+F", OnClick: e.find},
		{Label: "Find Next", Accel: "Ctrl+G", OnClick: e.findNext},
		{Label: "Replace All", Accel: "Ctrl+R", OnClick: e.replaceAll},
	})
	return e.menu.Init()
}

func (e *Editor) buildText() error {
	e.tv = &tk_widget.TextView{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: e.tk}, Parent: &e.frame.Window,
			X: 0, Y: barH, W: winW - sbW, H: winH - 2*barH,
		},
		Font: e.font,
	}
	return e.tv.Init()
}

func (e *Editor) buildScrollbar() error {
	e.sb = &tk_widget.Scrollbar{Window: tk_core.Window{
		Drawable: tk_core.Drawable{Conn: e.tk}, Parent: &e.frame.Window,
		X: winW - sbW, Y: barH, W: sbW, H: winH - 2*barH,
	}}
	return e.sb.Init()
}

func (e *Editor) buildStatus() error {
	gc, err := e.tk.CreateGC1(e.conn.DefaultBlackPixel(), e.conn.DefaultWhitePixel(), e.font.ID)
	if err != nil {
		return err
	}
	e.statusGc = gc
	e.status = &tk_widget.Label{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: e.tk}, Parent: &e.frame.Window,
			X: 0, Y: winH - barH, W: winW, H: barH,
			EventMask: base.CARD32(event_mask.Exposure),
		},
		Renderer: e.font,
		Gc:       e.statusGc.XID,
		Align:    tk_widget.AlignLeft,
	}
	return e.status.Init()
}

func (e *Editor) wireCallbacks() {
	e.tv.OnChange = func() { e.modified = true; e.refresh() }
	e.tv.OnScroll = func() { e.syncScrollbar() }
	// All hotkeys are menu accelerators; let the menu bar dispatch them.
	e.tv.OnKey = e.menu.HandleKey
	e.sb.OnScroll = func(top int) { e.tv.ScrollTo(top) }
	// closing the main window (WM title-bar) goes through the quit guard
	e.frame.OnClose = e.quit
}

func (e *Editor) setupClipboard() error {
	win, err := rpc.CreateWindow1(e.conn, e.conn.DefaultRoot(), -10, -10, 1, 1, clipboard.EventMask)
	if err != nil {
		return err
	}
	e.clipWin = win
	cb, err := clipboard.New(e.conn, win, "CLIPBOARD")
	if err != nil {
		return err
	}
	e.clip = cb
	e.conn.RegisterWindowHandler(win, cb)
	cb.OnPaste = func(s string) {
		if s == "" {
			return
		}
		e.tv.Insert(s)
		_ = e.tv.Draw()
		e.modified = true
		e.refresh()
	}
	return nil
}

// --- clipboard actions ---

func (e *Editor) copy() {
	if s := e.tv.SelectedText(); s != "" {
		if err := e.clip.Own(s); err != nil {
			log.Printf("copy: %v", err)
		}
	}
}

func (e *Editor) cut() {
	if e.tv.SelectedText() == "" {
		return
	}
	e.copy()
	e.tv.DeleteSelection()
	_ = e.tv.Draw()
	e.modified = true
	e.refresh()
}

func (e *Editor) paste() {
	if _, err := e.clip.RequestText(); err != nil { // result arrives via OnPaste
		log.Printf("paste: %v", err)
	}
}

// --- file actions ---

// open guards unsaved changes, then shows the file picker.
func (e *Editor) open() { e.guard(e.openPicker) }

// quit guards unsaved changes, then exits.
func (e *Editor) quit() { e.guard(func() { os.Exit(0) }) }

// newFile guards unsaved changes, then clears the buffer.
func (e *Editor) newFile() {
	e.guard(func() {
		e.tv.SetText("")
		e.filename = ""
		e.modified = false
		e.refresh()
		e.tv.Focus()
	})
}

// guard runs action immediately if the buffer is unmodified, otherwise asks the
// user to confirm discarding changes first.
func (e *Editor) guard(action func()) {
	if !e.modified {
		action()
		return
	}
	if e.confirm != nil {
		return
	}
	const cw, ch = 340, 90
	c := &dialog.Confirm{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: e.tk},
			X:        winX + (winW-cw)/2, Y: winY + (winH-ch)/2, W: cw, H: ch,
		},
		Font:     e.font,
		Message:  "Discard unsaved changes?",
		Floating: true,
		Title:    "Unsaved changes",
	}
	c.OnYes = func() { e.closeConfirm(); action() }
	c.OnNo = func() { e.closeConfirm() }
	if err := c.Init(); err != nil {
		return
	}
	_ = c.Raise()
	e.confirm = c
}

func (e *Editor) closeConfirm() {
	if e.confirm == nil {
		return
	}
	_ = e.confirm.Destroy()
	e.confirm = nil
	e.tv.Focus()
}

// openPicker shows the file picker and loads the chosen file.
func (e *Editor) openPicker() {
	if e.picker != nil {
		return
	}
	const pw, ph = 460, 360
	// A separate, window-manager-managed window centered over the editor
	// (X/Y are screen coordinates for a floating window).
	p := &dialog.FilePicker{
		Window: tk_core.Window{
			Drawable: tk_core.Drawable{Conn: e.tk},
			X:        winX + (winW-pw)/2, Y: winY + (winH-ph)/2, W: pw, H: ph,
		},
		Font:     e.font,
		Floating: true,
		Title:    "Open File",
	}
	p.OnAccept = func(path string) { e.closePicker(); e.loadFile(path) }
	p.OnCancel = func() { e.closePicker() }
	if err := p.Init(); err != nil {
		return
	}
	start, _ := filepath.Abs(".")
	if e.filename != "" {
		if d, err := filepath.Abs(filepath.Dir(e.filename)); err == nil {
			start = d
		}
	}
	_ = p.Open(start)
	_ = p.Raise()
	e.picker = p
}

func (e *Editor) closePicker() {
	if e.picker == nil {
		return
	}
	_ = e.picker.Destroy()
	e.picker = nil
	e.tv.Focus()
}

// loadFile reads path into the buffer.
func (e *Editor) loadFile(path string) {
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		e.flash(fmt.Sprintf("open failed: %v", err))
		return
	}
	e.tv.SetText(string(data))
	e.filename = path
	e.modified = false
	e.refresh()
}

// save writes to the current file, or asks for a name if there is none yet.
func (e *Editor) save() {
	if e.filename == "" {
		e.saveAs()
		return
	}
	e.writeFile(e.filename)
}

// saveAs prompts for a path and saves to it.
func (e *Editor) saveAs() {
	e.askFilename("Save as:", e.filename, func(path string) {
		if path == "" {
			return
		}
		e.filename = path
		e.writeFile(path)
	})
}

func (e *Editor) writeFile(path string) {
	if err := os.WriteFile(path, []byte(e.tv.Text()), 0o644); err != nil {
		e.flash(fmt.Sprintf("save failed: %v", err))
		return
	}
	e.modified = false
	e.flash(fmt.Sprintf("saved %s", path))
}

// --- find / replace ---

func (e *Editor) find() {
	e.askFilename("Find:", e.lastQuery, func(q string) {
		e.tv.Focus()
		if q == "" {
			return
		}
		e.lastQuery = q
		if !e.tv.FindNext(q) {
			e.flash("not found: " + q)
		}
	})
}

func (e *Editor) findNext() {
	if e.lastQuery == "" {
		e.find()
		return
	}
	if !e.tv.FindNext(e.lastQuery) {
		e.flash("not found: " + e.lastQuery)
	}
}

func (e *Editor) replaceAll() {
	e.askFilename("Replace - find:", e.lastQuery, func(q string) {
		if q == "" {
			e.tv.Focus()
			return
		}
		e.lastQuery = q
		e.askFilename("Replace with:", "", func(r string) {
			n := e.tv.ReplaceAll(q, r)
			e.tv.Focus()
			e.flash(fmt.Sprintf("replaced %d occurrence(s)", n))
		})
	})
}

// --- status / scrollbar refresh ---

func (e *Editor) refresh() {
	e.syncScrollbar()
	e.updateStatus()
}

func (e *Editor) syncScrollbar() {
	e.sb.SetRange(e.tv.LineCount(), e.tv.VisibleLines(), e.tv.TopLine())
}

func (e *Editor) updateStatus() {
	name := e.filename
	if name == "" {
		name = "(no file)"
	}
	flag := ""
	if e.modified {
		flag = " *"
	}
	_ = e.status.SetText(fmt.Sprintf("%s%s  -  %d lines", name, flag, e.tv.LineCount()))

	// reflect the file + modified state in the window title
	title := "go-xedit"
	if e.filename != "" {
		title += " - " + filepath.Base(e.filename)
	}
	_ = e.frame.SetName(title + flag)
}

// flash shows a transient message in the status line (until the next refresh).
func (e *Editor) flash(msg string) { _ = e.status.SetText(msg) }

// openMono opens a monospace font at px pixels, falling back to "fixed".
func openMono(conn *core.X11Conn, px int) *font.Font {
	if px > 0 {
		xlfd := fmt.Sprintf("-*-*-medium-r-normal-*-%d-*-*-*-m-*-iso8859-1", px)
		if f, err := font.Open(conn, xlfd); err == nil {
			return f
		}
	}
	f, err := font.Open(conn, "fixed")
	if err != nil {
		log.Fatalf("no usable font: %v", err)
	}
	return f
}
