// Command tabbed is a standalone XEmbed tab manager — a Go analogue of
// suckless' tabbed. It spawns N independent terminal processes (each a
// separate X client with its own connection) and re-parents their top-level
// windows into one container window.  Tabs are switched by show/hide; the tab
// manager never multiplexes terminals itself.
//
// Usage:  tabbed [terminal-command ...]
//
// Each positional argument is one terminal command to spawn (default:
// ./demo/terminal).  Tab switching: Alt+1..9 (direct), Alt+Tab (next),
// Alt+Shift+Tab (prev), Alt+Q (quit).
//
// The container window is override_redirect (unmanaged), so no window manager
// competes for reparenting.
package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/events/event_mask"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/keyboard"
	"github.com/X11Libre/go-x11proto/tk/xembed"
)

const (
	geomX = 50
	geomY = 50
	geomW = 960
	geomH = 600
)

// xaATOM is the predefined XA_ATOM property type.
const xaATOM base.ATOM = 4

// tab is one embedded terminal window.
type tab struct {
	win        base.WINDOW
	clientBase base.CARD32 // X resource-id base of the owning client
	visible    bool
}

// tabber glues the connection, XEmbed support and the tab list together.
type tabber struct {
	conn      *core.X11Conn
	xe        *xembed.Conn
	container base.WINDOW
	del       base.ATOM // WM_DELETE_WINDOW atom
	keyMap    *keyboard.Map

	clientMask base.CARD32 // setup RidMask — to group windows per client
	tabs       []tab
	active     int // index of the visible tab; -1 = none
}

func main() {
	conn, err := proto.Dial("")
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	xe, err := xembed.New(conn)
	if err != nil {
		log.Fatalf("xembed init: %v", err)
	}

	tb := &tabber{conn: conn, xe: xe, active: -1, clientMask: conn.Setup.RidMask}
	if err := tb.setup(conn); err != nil {
		log.Fatalf("setup: %v", err)
	}

	terminals := os.Args[1:]
	if len(terminals) == 0 {
		terminals = []string{"./demo/terminal"}
	}
	for _, t := range terminals {
		tb.spawn(t)
	}

	tb.run()
}

// setup creates the container window, selects events and maps it.
func (tb *tabber) setup(conn *core.X11Conn) error {
	root := conn.DefaultRoot()

	spec := rpc.WindowSpec{
		Parent:      root,
		X:           geomX,
		Y:           geomY,
		Width:       geomW,
		Height:      geomH,
		BorderWidth: 0,
		EventMask: base.CARD32(
			event_mask.KeyPress | event_mask.KeyRelease |
				event_mask.Exposure | event_mask.StructureNotify |
				event_mask.FocusChange,
		),
		SetBackPixel: true,
		BackPixel:    conn.DefaultBlackPixel(),
	}
	container, err := rpc.CreateWindow(conn, spec)
	if err != nil {
		return err
	}
	tb.container = container

	// Unmanaged so no WM reparents or decorates our container.
	if err := rpc.ChangeWindowAttributes(conn, &request.ChangeWindowAttributesRequest{
		Window:           container,
		ValueMask:        request.CW_OVERRIDE_REDIRECT,
		OverrideRedirect: 1,
	}); err != nil {
		return err
	}
	if err := rpc.SetWindowName(conn, container, "tabbed"); err != nil {
		return err
	}

	// Advertise WM_DELETE_WINDOW so Alt+Q can close gracefully.
	protoAtom, err := rpc.InternAtom(conn, "WM_PROTOCOLS")
	if err != nil {
		return err
	}
	del, err := rpc.InternAtom(conn, "WM_DELETE_WINDOW")
	if err != nil {
		return err
	}
	tb.del = del
	if err := rpc.ChangeProperty32(conn, 0, container, protoAtom, xaATOM,
		[]base.CARD32{base.CARD32(del)}); err != nil {
		return err
	}

	// Watch root's children so we see each terminal's top-level window
	// being created (CreateNotify with parent == root).
	if err := rpc.ChangeWindowAttributes(conn, &request.ChangeWindowAttributesRequest{
		Window:    root,
		ValueMask: request.CW_EVENT_MASK,
		EventMask: base.CARD32(event_mask.SubstructureNotify),
	}); err != nil {
		return err
	}

	// Load the keyboard map once for keysym resolution.
	tb.keyMap, err = keyboard.Load(conn)
	if err != nil {
		log.Printf("keyboard map unavailable: %v", err)
	}
	return rpc.MapWindow(conn, container)
}

// spawn launches one terminal command as a detached child process.
func (tb *tabber) spawn(cmdLine string) {
	cmd := exec.Command("/bin/sh", "-c", cmdLine)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		log.Printf("spawn %q: %v", cmdLine, err)
		return
	}
	go func() { _ = cmd.Wait() }()
}

// run is the main X11 event loop.
func (tb *tabber) run() {
	evCh := tb.conn.Events()
	for {
		ev, ok := <-evCh
		if !ok {
			return
		}
		switch e := ev.(type) {
		case *events.CreateEvent:
			// A new top-level window: probably a terminal. Embed it if we
			// don't already track it.
			if e.ParentWindow == tb.conn.DefaultRoot() {
				tb.maybeEmbed(e.TargetWindow)
			}
		case *events.ConfigureEvent:
			// Terminal resized externally; keep it filling the container.
			tb.fitActive()
		case *events.ExposeEvent:
			// nothing to draw beyond children; keep active terminal sized.
			tb.fitActive()
		case *events.FocusInEvent:
			tb.focusActive()
		case *events.KeyPressEvent:
			tb.onKey(e)
		case *events.ClientMessageEvent:
			if tk_core.IsWMDelete(ev, tb.del) {
				return
			}
		}
	}
}

// clientBaseOf returns the resource-id base shared by all windows of one X
// client (X resource ids encode the owning client in the high bits).
func (tb *tabber) clientBaseOf(win base.WINDOW) base.CARD32 {
	return base.CARD32(win) &^ tb.clientMask
}

// maybeEmbed re-parents a freshly created top-level window into the container
// and applies the XEmbed handshake, unless the window's client is already
// tracked. A terminal creates several top-level windows (the terminal window
// plus clipboard helper windows); we only embed the first one per client.
func (tb *tabber) maybeEmbed(win base.WINDOW) {
	cb := tb.clientBaseOf(win)
	for _, t := range tb.tabs {
		if t.clientBase == cb {
			return
		}
	}
	if err := tb.xe.Embed(tb.container, win, 0, 0, geomW, geomH, 0); err != nil {
		log.Printf("embed %#x: %v", win, err)
		return
	}
	tb.tabs = append(tb.tabs, tab{win: win, clientBase: cb})
	if tb.active < 0 {
		// First tab: show it immediately.
		tb.active = 0
		tb.show()
	} else {
		// Embed() maps the window; a non-first tab must stay hidden until it
		// is switched to.
		_ = rpc.UnmapWindow(tb.conn, win)
	}
}

// show makes tp the visible tab and hides every other one.
func (tb *tabber) show() {
	if tb.active < 0 || tb.active >= len(tb.tabs) {
		return
	}
	target := tb.tabs[tb.active].win
	for i := range tb.tabs {
		if tb.tabs[i].win == target {
			continue
		}
		if tb.tabs[i].visible {
			_ = tb.xe.Deactivate(tb.container, tb.tabs[i].win, 0)
			_ = rpc.UnmapWindow(tb.conn, tb.tabs[i].win)
			tb.tabs[i].visible = false
		}
	}
	_ = rpc.ConfigureWindow(tb.conn, &request.ConfigureWindowRequest{
		Window: target,
		ValueMask: request.CONFIG_WINDOW_X | request.CONFIG_WINDOW_Y |
			request.CONFIG_WINDOW_WIDTH | request.CONFIG_WINDOW_HEIGHT,
		X: 0, Y: 0, Width: geomW, Height: geomH,
	})
	_ = rpc.MapWindow(tb.conn, target)
	_ = tb.xe.Activate(tb.container, target, 0)
	tb.tabs[tb.active].visible = true
	tb.focusActive()
}

// fitActive re-applies the container geometry to the active terminal.
func (tb *tabber) fitActive() {
	if tb.active < 0 || tb.active >= len(tb.tabs) {
		return
	}
	win := tb.tabs[tb.active].win
	_ = rpc.ConfigureWindow(tb.conn, &request.ConfigureWindowRequest{
		Window:    win,
		ValueMask: request.CONFIG_WINDOW_WIDTH | request.CONFIG_WINDOW_HEIGHT,
		Width:     geomW,
		Height:    geomH,
	})
}

// focusActive forwards keyboard focus to the active embedded terminal.
func (tb *tabber) focusActive() {
	if tb.active < 0 || tb.active >= len(tb.tabs) {
		return
	}
	_ = tb.xe.FocusIn(tb.container, tb.tabs[tb.active].win, 0, xembed.FocusCurrent)
}

// onKey handles the Alt-based tab-switching shortcuts.
func (tb *tabber) onKey(ev *events.KeyPressEvent) {
	if tb.keyMap == nil {
		return
	}
	k := tb.keyMap.Lookup(ev.Key, ev.State)
	switch {
	case ev.State&0x8 != 0 && k.Keysym >= '1' && k.Keysym <= '9':
		// Alt+1..9 → direct tab.
		target := int(k.Keysym - '1')
		if target < len(tb.tabs) {
			tb.active = target
			tb.show()
		}
	case ev.State&0x8 != 0 && k.Keysym == 0xff09: // Alt+Tab
		n := len(tb.tabs)
		if n > 1 {
			if k.Shift {
				tb.active = (tb.active - 1 + n) % n
			} else {
				tb.active = (tb.active + 1) % n
			}
			tb.show()
		}
	case ev.State&0x8 != 0 && (k.Keysym == 'q' || k.Keysym == 'Q'): // Alt+Q
		os.Exit(0)
	}
}
