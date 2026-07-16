// Command terminal-aa-detach is a terminal emulator that can detach from the
// X server and later reattach, controlled via signals, an inherited control
// fd (TERM_CTRL_FD), or (optionally) a named control pipe + registry.
//
// Signals:
//
//	SIGUSR1   detach
//	SIGUSR2   attach to $DISPLAY
//	SIGINT    quit (kills shell)
//	SIGTERM   quit (kills shell)
//
// Inherited control fd (TERM_CTRL_FD, set by the parent that spawned us):
//
//	detach            detach from X server (shell keeps running)
//	attach <display>  attach to display (e.g. ":1"); empty = $DISPLAY
//	status            print "attached" or "detached"
//	quit              exit (kills shell)
//
// Named control pipe (--pipe PATH): a FIFO is created at PATH and the terminal
// is registered under --name so other processes can drive it via
// termctl.Open(name). Same command set as above.
//
// Usage:
//
//	terminal-aa-detach [--detached] [--pipe PATH] [--name NAME] [shell-command]
//
// With --detached, the terminal starts without an X11 connection. The shell
// runs in a PTY with no visible window until "attach <display>" is sent.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/X11Libre/go-x11proto/tk/term/termctl"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix(fmt.Sprintf("[pid %d] ", os.Getpid()))
	log.Printf("START args=%v", os.Args)

	var (
		startDetached bool
		pipePath      string
		name          string
		shell         string
	)
	// Parse --detached / --pipe PATH / --name NAME plus an optional trailing
	// shell command (kept legible for a demo).
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--detached":
			startDetached = true
		case "--pipe":
			if i+1 < len(args) {
				pipePath = args[i+1]
				i++
			}
		case "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		default:
			if shell == "" && len(args[i]) > 0 && args[i][0] != '-' {
				shell = args[i]
			}
		}
	}

	opts := []termctl.Opt{}
	if shell != "" {
		opts = append(opts, termctl.WithShell(shell))
	}
	if name != "" {
		opts = append(opts, termctl.WithName(name))
	}
	if pipePath != "" {
		opts = append(opts, termctl.WithControlPipe(pipePath))
	}
	// Inherited control fd from a parent (TERM_CTRL_FD convention).
	if fdStr := os.Getenv("TERM_CTRL_FD"); fdStr != "" {
		var fd int
		if _, err := fmt.Sscanf(fdStr, "%d", &fd); err != nil {
			log.Fatalf("invalid TERM_CTRL_FD: %v", err)
		}
		opts = append(opts, termctl.WithControlFD(fd))
	}

	h, err := termctl.New(opts...)
	if err != nil {
		log.Fatalf("termctl.New: %v", err)
	}

	go signalLoop(h)

	if !startDetached {
		if err := h.Attach(""); err != nil {
			log.Fatalf("initial attach: %v", err)
		}
		log.Printf("main: initial attach done")
	}

	// Block until the shell exits (Run also handles cleanup of the control
	// channel + registry entry).
	if err := h.Run(); err != nil {
		log.Printf("run: %v", err)
	}
}

// signalLoop runs in a dedicated OS thread so signal delivery is never blocked
// by the X event loop's (possibly cgo-backed) read elsewhere.
func signalLoop(h *termctl.TermHandle) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	log.Printf("signalLoop started; Notify registered")
	for sig := range sigCh {
		log.Printf("SIGNAL received: %v", sig)
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			_ = h.Close()
			os.Exit(0)
		case syscall.SIGUSR1:
			if h.IsAttached() {
				if err := h.Detach(); err != nil {
					log.Printf("SIGUSR1 detach: %v", err)
				} else {
					log.Printf("SIGUSR1 detached")
				}
			}
		case syscall.SIGUSR2:
			if !h.IsAttached() {
				if err := h.Attach(""); err != nil {
					log.Printf("SIGUSR2 attach: %v", err)
				} else {
					log.Printf("SIGUSR2 attached")
				}
			}
		}
	}
}
