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
	"strings"
	"syscall"

	"github.com/X11Libre/go-x11proto/proto"
	"github.com/X11Libre/go-x11proto/proto/base"
	proto_core "github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
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
	sigCh    chan os.Signal
	ctrlF    *os.File // control pipe for responses

	// remembered geometry across attach/detach cycles
	winW, winH base.CARD16
	winX, winY base.INT16
}

func main() {
	log.SetFlags(0)

	a := &app{
		cmdCh: make(chan string, 8),
		sigCh: make(chan os.Signal, 1),
		winW:  800,
		winH:  480,
		winX:  50,
		winY:  50,
	}
	signal.Notify(a.sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

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

	if !startDetached {
		if err := a.doAttach(""); err != nil {
			log.Fatalf("initial attach: %v", err)
		}
	}

	if err := a.eventLoop(); err != nil {
		log.Fatalf("run: %v", err)
	}
}

func (a *app) ctrlReader(f *os.File) {
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		a.cmdCh <- strings.TrimSpace(sc.Text())
	}
}

func (a *app) eventLoop() error {
	var evCh <-chan events.Event
	if a.attached {
		evCh = a.conn.Events()
	}

	for {
		var dirtyCh <-chan struct{}
		if a.t != nil {
			dirtyCh = a.t.Dirty()
		}

		select {
		case ev, ok := <-evCh:
			if !ok {
				evCh = nil
				continue
			}
			a.conn.DeliverWindowEvent(ev)

		case <-dirtyCh:
			if a.attached {
				_ = a.t.Draw()
			}

		case sig := <-a.sigCh:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				return nil
			case syscall.SIGUSR1:
				if a.attached {
					a.doDetach()
					evCh = nil
				}
			case syscall.SIGUSR2:
				if !a.attached {
					if err := a.doAttach(""); err != nil {
						log.Printf("SIGUSR2 attach: %v", err)
					} else {
						evCh = a.conn.Events()
					}
				}
			}

		case line := <-a.cmdCh:
			a.handleCmd(line)
			if a.attached {
				evCh = a.conn.Events()
			} else {
				evCh = nil
			}
		}
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
	// Remember current window geometry before destroying resources.
	if g, err := a.t.Window.GetGeometry(); err == nil {
		a.winW = g.Width
		a.winH = g.Height
		a.winX = g.X
		a.winY = g.Y
	}
	_ = a.t.Detach()
	if a.face != nil {
		a.face.Close()
		a.face = nil
	}
	a.tk = tk_core.TkConn{}
	a.rdr = nil
	// Sync: wait for all pending X requests (DestroyWindow etc.) to be
	// sent/processed before closing the connection.
	_, _ = rpc.GetGeometry(a.conn, a.conn.DefaultRoot())
	a.conn.Close()
	a.conn = nil
	a.attached = false
	log.Printf("detached from X server")
}

func (a *app) doAttach(display string) error {
	if a.attached {
		return fmt.Errorf("already attached")
	}

	conn, err := proto.DialBE(display)
	if err != nil {
		return fmt.Errorf("dial %q: %w", display, err)
	}

	tk := tk_core.MakeTkConn(conn)

	rdr, err := tk_render.Open(&tk)
	if err != nil {
		conn.Close()
		return fmt.Errorf("RENDER: %w", err)
	}

	face, err := ttf.Open(&tk, rdr, ttfPath, 13, 96)
	if err != nil {
		conn.Close()
		return fmt.Errorf("open TTF: %w", err)
	}

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

	if err := a.t.Attach(&tk, tk.GetRoot().XID); err != nil {
		face.Close()
		conn.Close()
		a.conn = nil
		return err
	}

	a.attached = true
	log.Printf("attached to display %q", display)
	return nil
}
