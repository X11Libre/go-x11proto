package termctl

import (
	"fmt"
	"log"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
	"github.com/X11Libre/go-x11proto/tk/font/ttf"
)

// Attach opens an X window on the given display ("" means $DISPLAY) and
// renders the running shell there. If already attached, Attach detaches
// first. It is safe to call from any goroutine.
func (h *TermHandle) Attach(display string) error {
	h.mu.Lock()
	if h.attached {
		h.mu.Unlock()
		if err := h.Detach(); err != nil {
			return err
		}
		h.mu.Lock()
	}
	if h.t == nil {
		h.mu.Unlock()
		return fmt.Errorf("termctl: handle has no shell (not started)")
	}

	// Fresh per-cycle event-loop channels (the previous cycle's were consumed
	// by its stop/detach).
	h.runLoopStop = make(chan struct{})
	h.runLoopWait = make(chan struct{})

	conn, err := proto.DialBE(display)
	if err != nil {
		h.mu.Unlock()
		return fmt.Errorf("dial %q: %w", display, err)
	}
	tk := tk_core.MakeTkConn(conn)
	rdr, err := tk_render.Open(&tk)
	if err != nil {
		conn.Close()
		h.mu.Unlock()
		return fmt.Errorf("RENDER: %w", err)
	}
	face, err := ttf.Open(&tk, rdr, h.ttfPath, 13, 96)
	if err != nil {
		conn.Close()
		h.mu.Unlock()
		return fmt.Errorf("open TTF: %w", err)
	}

	h.t.AAFace = face
	h.t.AARender = rdr
	h.t.Fg, h.t.Bg = 0, 0
	h.t.W = base.CARD16(h.geom.W)
	h.t.H = base.CARD16(h.geom.H)
	h.t.X = base.INT16(h.geom.X)
	h.t.Y = base.INT16(h.geom.Y)
	h.t.Name = h.title
	h.t.SetBackPixel = true
	h.t.BackPixel = conn.DefaultBlackPixel()

	if err := h.t.Attach(&tk, tk.GetRoot().XID); err != nil {
		face.Close()
		conn.Close()
		h.mu.Unlock()
		return err
	}

	h.conn = xconn{conn: conn, tk: tk, rdr: rdr, face: face}
	h.attached = true
	h.connSeq++
	seq := h.connSeq
	h.mu.Unlock()

	// Draw the produced shell output immediately.
	_ = h.t.Draw()
	h.t.Dirty()

	// Drive the event loop for this connection in its own goroutine. When the
	// connection dies (window closed), it auto-detaches; on explicit Detach we
	// stop the loop via runLoopStop.
	go h.runLoop(conn, seq)
	return nil
}

// Detach tears down the X window but keeps the shell running. Safe to call
// from any goroutine; a no-op when already detached.
func (h *TermHandle) Detach() error {
	h.mu.Lock()
	if !h.attached {
		h.mu.Unlock()
		return nil
	}
	// Best-effort geometry capture before tearing down the connection.
	if g, err := h.t.Window.GetGeometry(); err == nil {
		h.geom.W = uint16(g.Width)
		h.geom.H = uint16(g.Height)
		h.geom.X = int16(g.X)
		h.geom.Y = int16(g.Y)
	}
	// Snapshot the live X handles and mark detached, then release the lock
	// BEFORE stopping the run loop: runLoop needs h.mu (for autoDetachDead)
	// and would deadlock if detach held it while waiting on runLoopWait.
	conn := h.conn.conn
	face := h.conn.face
	h.conn.face = nil
	h.conn.tk = tk_core.TkConn{}
	h.conn.rdr = nil
	h.conn.conn = nil
	h.attached = false
	h.mu.Unlock()

	// term.Detach must run while the X connection is still alive (it issues
	// Destroy/Free requests). Do it first, then close the connection so the
	// blocked RunLoop goroutine receives EOF and exits (closing runLoopWait).
	if err := h.t.Detach(); err != nil {
		log.Printf("termctl: Detach: %v", err)
	}
	if conn != nil {
		conn.Close()
	}
	h.stopRunLoop()

	if face != nil {
		face.Close()
	}
	return nil
}

// runLoop drives term.RunLoop on the given connection. It returns when the
// connection closes (user closed window / server disconnected) or when
// stopRunLoop is called. On an unexpected close it auto-detaches so the shell
// survives.
func (h *TermHandle) runLoop(conn *proto_core.X11Conn, seq int) {
	defer close(h.runLoopWait)
	for {
		// Terminate early on explicit stop.
		select {
		case <-h.runLoopStop:
			return
		default:
		}
		h.t.RunLoop(conn)
		// RunLoop returned: either we were asked to stop, or the connection
		// died. Distinguish via the seq guard.
		h.mu.Lock()
		stopped := h.connSeq != seq
		if !stopped && h.attached {
			// Connection died on its own: auto-detach without touching the
			// (now dead) socket.
			h.autoDetachDead()
		}
		h.mu.Unlock()
		if stopped {
			return
		}
		// Dead connection: nothing more to drive until re-attach.
		return
	}
}

// autoDetachDead assumes h.mu is held and the connection is already gone.
func (h *TermHandle) autoDetachDead() {
	h.t.ResetForReattach()
	h.conn.face = nil
	h.conn.tk = tk_core.TkConn{}
	h.conn.rdr = nil
	h.conn.conn = nil
	h.attached = false
	log.Printf("termctl: X connection lost, auto-detached (shell still running)")
}

// Stop terminates the shell (and detaches the window first, if attached).
func (h *TermHandle) Stop() error {
	h.mu.Lock()
	attached := h.attached
	t := h.t
	h.mu.Unlock()
	if attached {
		_ = h.Detach()
	}
	if t == nil {
		return nil
	}
	return t.Stop()
}

// Close stops the shell (if running) and releases all resources: control
// pipe, registry entry, and the OnExit cleanup callback. It is safe to call
// multiple times.
func (h *TermHandle) Close() error {
	h.Stop()
	h.mu.Lock()
	t := h.t
	h.mu.Unlock()
	if t != nil {
		_ = t.Stop()
	}
	if h.ctrl != nil {
		h.ctrl.close()
		h.ctrl = nil
	}
	unregister(h.name)
	if h.onExit != nil {
		h.onExit()
	}
	return nil
}

// Run blocks until the shell has exited, then performs cleanup (control pipe,
// registry, OnExit). It is the typical entry point for a process that only
// spawns and owns the terminal.
func (h *TermHandle) Run() error {
	h.mu.Lock()
	t := h.t
	h.mu.Unlock()
	if t == nil {
		return fmt.Errorf("termctl: handle has no shell (not started)")
	}
	done := make(chan struct{})
	t.OnExit = func(error) {
		select {
		case <-done:
		default:
			close(done)
		}
	}
	<-done
	return h.Close()
}
