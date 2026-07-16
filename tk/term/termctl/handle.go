// Package termctl provides an embeddable, detachable terminal emulator
// handle built on top of tk/term. It is meant to be driven from another Go
// program (such as starfleetctl) that wants to spawn background terminals,
// optionally show them on an X display, detach them again, and later
// re-attach — without tmux in between.
//
// A TermHandle owns a shell running in a PTY. The X window is created and
// destroyed on demand via Attach/Detach; the shell keeps running while
// detached. A handle may also expose a control pipe so a *different* process
// can attach/detach/stop it by name (see WithControlPipe and Open).
package termctl

import (
	"fmt"
	"sync"

	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
	"github.com/X11Libre/go-x11proto/tk/font/ttf"
	"github.com/X11Libre/go-x11proto/tk/term"
)

// DefaultTTFPath is the font used for the antialiased text path when the
// caller does not override it.
const DefaultTTFPath = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

// DefaultTitle is the window title used when the caller does not set one.
const DefaultTitle = "termctl"

// TermHandle is a detachable terminal with a shell running in a PTY. The X
// window is optional and created on Attach.
type TermHandle struct {
	mu sync.Mutex

	t   *term.Term

	name string
	id   string // stable id for the lifetime of the handle

	// X connection state.
	attached bool
	conn     xconn
	connSeq  int // bumped on every successful attach (EOF stale-guard)

	// runLoopStop is closed to ask the current event-loop goroutine to exit.
	// runLoopWait is closed by the event-loop goroutine when it has actually
	// exited; detach() waits on it. Both are recreated on each Attach.
	runLoopStop chan struct{}
	runLoopWait chan struct{}

	// configuration
	shell    string
	extraEnv []string
	title    string
	ttfPath  string
	geom     Geometry
	onExit   func()

	// shellExited is set by onShellExit (term.Start's wait goroutine).
	shellExited bool

	// control channel (optional): FIFO pipe or an inherited fd.
	ctrl controller
}

// Geometry describes the desired window size and position.
type Geometry struct {
	W, H uint16
	X, Y int16
}

// xconn bundles the X11 connection and the derived tk/render/ttf handles so a
// single attach/detach cycle can be torn down together.
type xconn struct {
	conn *proto_core.X11Conn
	tk   tk_core.TkConn
	rdr  *tk_render.Render
	face *ttf.Face
}

// New creates a TermHandle and starts the shell immediately (detached, no
// window). Use Attach to show it on a display. Call Run (or Close) to wait
// for and clean up after the shell.
func New(opts ...Opt) (*TermHandle, error) {
	h := &TermHandle{
		id:   newID(),
		name: "",
		title: DefaultTitle,
		ttfPath: DefaultTTFPath,
		geom:   Geometry{W: 800, H: 480, X: 50, Y: 50},
		runLoopStop: make(chan struct{}),
		runLoopWait: make(chan struct{}),
	}
	for _, o := range opts {
		o(h)
	}
	if h.name == "" {
		h.name = h.id
	}

	if err := h.startShell(); err != nil {
		return nil, err
	}
	// Wire up the optional control channel (FIFO pipe or inherited fd). For
	// FIFO mode the name is registered so other processes can Open() it.
	if h.ctrl != nil {
		if err := h.ctrl.open(); err != nil {
			_ = h.Close()
			return nil, fmt.Errorf("control channel: %w", err)
		}
		if fc, ok := h.ctrl.(*fifoCtrl); ok {
			if err := register(h.name, fc.path); err != nil {
				_ = h.Close()
				return nil, err
			}
		}
	}
	return h, nil
}

// ID returns the stable identifier for this handle (set at New).
func (h *TermHandle) ID() string { return h.id }

// Name returns the caller-assigned name (defaults to ID when unset).
func (h *TermHandle) Name() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.name
}

// IsAttached reports whether the terminal currently has an X window.
func (h *TermHandle) IsAttached() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.attached
}

// stopRunLoop asks the current X event-loop goroutine to exit and waits until
// it has done so. It is a no-op if no loop is running (runLoopWait already
// closed). Safe to call multiple times.
func (h *TermHandle) stopRunLoop() {
	select {
	case <-h.runLoopStop:
		// already asked to stop
	default:
		close(h.runLoopStop)
	}
	<-h.runLoopWait
}
