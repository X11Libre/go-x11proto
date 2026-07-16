// Command terminal-aa-detach is a terminal emulator that can detach from the
// X server and later reattach, controlled via a pipe or signal.
//
// Control pipe (TERM_CTRL_FD):
//
//	detach            detach from X server (shell keeps running)
//	attach <display>  attach to display (e.g. ":1")
//	status            print "attached" or "detached"
//	quit              exit (kills shell)
//
// Signals:
//
//	SIGUSR1   detach
//	SIGUSR2   attach to $DISPLAY
//	SIGINT    quit
//	SIGTERM   quit
//
// Usage: terminal-aa-detach [--detached] [shell-command]
//
// With --detached, the terminal starts without an X11 connection. The shell
// runs in a PTY with no visible window until "attach <display>" is sent.
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/font/ttf"
	tk_render "github.com/X11Libre/go-x11proto/tk/render"
	"github.com/X11Libre/go-x11proto/tk/term"
)

const ttfPath = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"

type app struct {
	t    *term.Term
	conn *proto_core.X11Conn
	tk   tk_core.TkConn
	rdr  *tk_render.Render
	face *ttf.Face

	attached bool
	cmdCh    chan string
	ctrlF    *os.File // control pipe for responses

	// ctrlMu serialises attach/detach against the event loop's use of the X
	// connection. Signal/pipe handling runs in its own goroutines (see
	// ctrlLoop) so a busy X event stream can never starve detach/attach —
	// previously the single select in eventLoop let evCh win every round and
	// SIGUSR1/SIGUSR2/pipe commands were never processed while attached.
	ctrlMu sync.Mutex

	// attachSeq increments on every successful attach so the event loop can
	// tell whether a closed evCh belongs to the current connection or to an
	// already-replaced one (avoids detaching the new conn on a stale EOF).
	attachSeq int

	// remembered geometry across attach/detach cycles
	winW, winH base.CARD16
	winX, winY base.INT16
}

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix(fmt.Sprintf("[pid %d] ", os.Getpid()))
	log.Printf("START args=%v", os.Args)

	a := &app{
		cmdCh: make(chan string, 8),
		winW:  800,
		winH:  480,
		winX:  50,
		winY:  50,
	}

	if fdStr := os.Getenv("TERM_CTRL_FD"); fdStr != "" {
		var fd int
		if _, err := fmt.Sscanf(fdStr, "%d", &fd); err != nil {
			log.Fatalf("invalid TERM_CTRL_FD: %v", err)
		}
		f := os.NewFile(uintptr(fd), "ctrl")
		if f == nil {
			log.Fatalf("invalid control fd %d", fd)
		}
		a.ctrlF = f
		go a.ctrlReader(f)
	}

	startDetached := false
	shell := ""
	for _, arg := range os.Args[1:] {
		if arg == "--detached" {
			startDetached = true
		} else if shell == "" {
			shell = arg
		}
	}

	a.t = &term.Term{
		Type:    term.XTerm256Color,
		FgRGB:   [3]byte{0xff, 0xff, 0xff},
		BgRGB:   [3]byte{0x00, 0x00, 0x00},
		OnTitle: func(s string) { log.Printf("title: %s", s) },
		OnExit:  func(err error) { os.Exit(0) },
		Shell:   shell,
	}
	if err := a.t.InitTerm(); err != nil {
		log.Fatalf("init term: %v", err)
	}
	if err := a.t.Start(); err != nil {
		log.Fatalf("start shell: %v", err)
	}

	go a.signalLoop()
	go a.ctrlLoop()
	// xRunLoop drives the Term's event loop on the *current* X connection.
	// It is restarted on each attach/detach so it always uses the live conn.
	go a.xRunLoop()

	if !startDetached {
		if err := a.doAttach(""); err != nil {
			log.Fatalf("initial attach: %v", err)
		}
		log.Printf("main: initial attach done, attached=%v", a.attached)
	}

	select {} // run forever; control via signals/pipe
}

// signalLoop runs in a dedicated OS thread so signal delivery is never
// blocked by the X event loop's (possibly cgo-backed) read in another thread.
// Some go-x11proto paths block a thread in a C call (e.g. font/render), which
// can starve Go's default signal-delivery thread; pinning the handler to its
// own locked thread keeps detach/attach responsive while attached.
func (a *app) signalLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	log.Printf("signalLoop started; Notify registered")
	for sig := range sigCh {
		log.Printf("SIGNAL received: %v", sig)
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			os.Exit(0)
		case syscall.SIGUSR1:
			log.Printf("SIGUSR1: attached=%v", a.attached)
			a.ctrlMu.Lock()
			if a.attached {
				a.doDetach()
				log.Printf("SIGUSR1: doDetach done, attached=%v", a.attached)
			}
			a.ctrlMu.Unlock()
		case syscall.SIGUSR2:
			log.Printf("SIGUSR2: attached=%v", a.attached)
			a.ctrlMu.Lock()
			if !a.attached {
				if err := a.doAttach(""); err != nil {
					log.Printf("SIGUSR2 attach: %v", err)
				} else {
					log.Printf("SIGUSR2 attached ok")
				}
			}
			a.ctrlMu.Unlock()
		}
	}
}

func (a *app) ctrlReader(f *os.File) {
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		a.cmdCh <- strings.TrimSpace(sc.Text())
	}
}

// xRunLoop drives the Term's event loop on the current X connection using
// the library's own term.RunLoop (which correctly combines conn.Events(),
// DeliverWindowEvent and Dirty-driven redraws). It runs as a long-lived
// goroutine: while attached it runs RunLoop on the live conn; when the
// connection dies (RunLoop returns on EOF) it auto-detaches so the shell
// survives, then waits until the next attach restarts the loop.
func (a *app) xRunLoop() {
	for {
		a.ctrlMu.Lock()
		attached := a.attached
		conn := a.conn
		a.ctrlMu.Unlock()
		if !attached || conn == nil {
			// Idle until signalLoop/ctrlLoop flip attached=true (doAttach).
			time.Sleep(50 * time.Millisecond)
			continue
		}
		// Run the library event loop for this connection. It returns when the
		// connection closes (user closed window / server disconnected).
		a.t.RunLoop(conn)
		// Connection gone: auto-detach and keep the shell alive.
		a.ctrlMu.Lock()
		if a.attached {
			seq := a.attachSeq
			a.ctrlMu.Unlock()
			// Only detach if this is still the same connection cycle.
			a.ctrlMu.Lock()
			if a.attached && a.attachSeq == seq {
				a.detachOnDeadConn()
			}
			a.ctrlMu.Unlock()
		} else {
			a.ctrlMu.Unlock()
		}
	}
}

// ctrlLoop owns all attach/detach transitions driven by the control pipe.
// Signal handling lives in signalLoop (its own locked OS thread) so a busy X
// event loop can never starve detach/attach.
func (a *app) ctrlLoop() {
	for line := range a.cmdCh {
		a.ctrlMu.Lock()
		a.handleCmd(line)
		a.ctrlMu.Unlock()
	}
}

func (a *app) handleCmd(cmd string) {
	a.respond("> %s", cmd)
	switch {
	case cmd == "quit":
		os.Exit(0)

	case cmd == "detach":
		if !a.attached {
			a.respond("already detached")
			return
		}
		a.doDetach()
		a.respond("detached")

	case strings.HasPrefix(cmd, "attach "):
		if a.attached {
			a.respond("already attached")
			return
		}
		display := strings.TrimSpace(cmd[len("attach "):])
		if err := a.doAttach(display); err != nil {
			a.respond("attach failed: %v", err)
			log.Printf("attach %q: %v", display, err)
		} else {
			a.respond("attached to %s", display)
		}

	case cmd == "status":
		if a.attached {
			a.respond("attached")
		} else {
			a.respond("detached")
		}
	}
}

func (a *app) respond(format string, args ...any) {
	if a.ctrlF != nil {
		fmt.Fprintf(a.ctrlF, format+"\n", args...)
	}
}

func (a *app) doDetach() {
	if !a.attached {
		return
	}
	// Best-effort geometry capture; the connection may already be dead (e.g.
	// the user closed the window), in which case GetGeometry fails and we keep
	// the last known geometry.
	if a.conn != nil {
		if g, err := a.t.Window.GetGeometry(); err == nil {
			a.winW = g.Width
			a.winH = g.Height
			a.winX = g.X
			a.winY = g.Y
		}
	}
	// Detaching and closing may touch a connection that already errored out;
	// guard so a dead socket cannot panic and kill the shell.
	func() {
		defer func() { _ = recover() }()
		_ = a.t.Detach()
		if a.face != nil {
			a.face.Close()
			a.face = nil
		}
		a.tk = tk_core.TkConn{}
		a.rdr = nil
		a.conn.Close()
	}()
	a.conn = nil
	a.attached = false
	log.Printf("detached from X server")
}

// detachOnDeadConn handles the case where the X connection went away on its
// own (typically the user closing the window, which makes the X server drop
// the connection). Unlike doDetach it must NOT touch the connection or send
// any X requests — the socket is already gone and would panic — it only clears
// our state so the shell keeps running detached and a later attach can
// reconnect on a fresh connection.
func (a *app) detachOnDeadConn() {
	if !a.attached {
		return
	}
	// The X connection is already dead, so we must NOT call t.Detach() — its
	// Destroy/Free requests and t.Conn deref would panic on the dead socket.
	// Just drop our references; a.t's cached AA/RENDER/GC state is rebuilt by
	// the next term.Attach (initAA reassigns those fields). The window XID is
	// cleared so a later attach builds a fresh window.
	// The X connection is already dead, so we must NOT call t.Detach() — its
	// Destroy/Free requests would panic on the dead socket. ResetForReattach
	// clears the Term's cached X resources without any X request, so the next
	// term.Attach rebuilds everything fresh on a new connection.
	a.t.ResetForReattach()
	a.face = nil
	a.tk = tk_core.TkConn{}
	a.rdr = nil
	a.conn = nil
	a.attached = false
	log.Printf("X connection lost: auto-detached (shell still running)")
}

func (a *app) doAttach(display string) error {
	if a.attached {
		return fmt.Errorf("already attached")
	}
	log.Printf("doAttach: dialing %q", display)

	conn, err := proto.DialBE(display)
	if err != nil {
		return fmt.Errorf("dial %q: %w", display, err)
	}
	log.Printf("doAttach: dialed ok")

	tk := tk_core.MakeTkConn(conn)

	rdr, err := tk_render.Open(&tk)
	if err != nil {
		conn.Close()
		return fmt.Errorf("RENDER: %w", err)
	}
	log.Printf("doAttach: RENDER open ok")

	face, err := ttf.Open(&tk, rdr, ttfPath, 13, 96)
	if err != nil {
		conn.Close()
		return fmt.Errorf("open TTF: %w", err)
	}
	log.Printf("doAttach: TTF open ok")

	a.conn = conn
	a.tk = tk
	a.rdr = rdr
	a.face = face

	a.t.AAFace = face
	a.t.AARender = rdr
	a.t.Fg, a.t.Bg = 0, 0
	a.t.W = a.winW
	a.t.H = a.winH
	a.t.X = a.winX
	a.t.Y = a.winY
	a.t.Name = "terminal-aa-detach"
	a.t.SetBackPixel = true
	a.t.BackPixel = conn.DefaultBlackPixel()

	log.Printf("doAttach: calling term.Attach")
	if err := a.t.Attach(&tk, tk.GetRoot().XID); err != nil {
		face.Close()
		conn.Close()
		a.conn = nil
		return err
	}
	log.Printf("doAttach: term.Attach ok")

	a.attached = true
	a.attachSeq++
	// Draw the already-produced shell output immediately (the window was just
	// mapped by term.Attach), and also mark dirty so the event loop redraws on
	// the next cycle / Expose.
	_ = a.t.Draw()
	a.t.Dirty()
	log.Printf("attached to display %q", display)
	return nil
}
