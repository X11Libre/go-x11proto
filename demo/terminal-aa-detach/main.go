// Command terminal-aa-detach is a terminal emulator that can detach from the
// X server and later reattach, controlled via signals, an inherited control
// fd (TERM_CTRL_FD), or a named control pipe.
//
// It also serves as a stand-in for starfleetctl's caller side: it maintains a
// tiny name->pipe registry (a plain file) so a separate "ctl" invocation can
// drive an already-running terminal by name — exactly what starfleetctl will
// do later, just with its own registry instead of this demo's.
//
// Usage:
//
//	terminal-aa-detach run [--detached] [--name NAME] [--pipe PATH] [shell]
//	    Start a terminal. If --pipe is omitted a path is derived from --name
//	    (or a generated id) under the work temp dir. The (name, pipe) pair is
//	    recorded in the demo registry.
//
//	terminal-aa-detach ctl <name> <attach|detach|stop> [display]
//	    Drive a running terminal by name via OpenPipe.
//
// Signals (only relevant to a "run" process):
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
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/X11Libre/go-x11proto/tk/term/termctl"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix(fmt.Sprintf("[pid %d] ", os.Getpid()))

	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "run":
		cmdRun(args[1:])
	case "ctl":
		cmdCtl(args[1:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  terminal-aa-detach run [--detached] [--name NAME] [--pipe PATH] [shell]
  terminal-aa-detach ctl <name> <attach|detach|stop> [display]`)
}

// workDir returns the demo's registry directory (prefers MPBT_WORK_TMPDIR).
func workDir() string {
	if d := os.Getenv("MPBT_WORK_TMPDIR"); d != "" {
		return d
	}
	return os.TempDir()
}

// registryPath is the demo's tiny name->pipe store (stand-in for starfleetctl).
func registryPath() string {
	return workDir() + "/termctl-demo-registry.txt"
}

// regGet looks up a name in the demo registry and returns its pipe path.
func regGet(name string) (string, bool) {
	f, err := os.Open(registryPath())
	if err != nil {
		return "", false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && parts[0] == name {
			return parts[1], true
		}
	}
	return "", false
}

// regPut records name->pipe in the demo registry.
func regPut(name, pipe string) {
	path := registryPath()
	var b strings.Builder
	b.WriteString("# termctl demo registry: name=pipe\n")
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
				continue
			}
			if strings.HasPrefix(line, name+"=") {
				continue // overwrite existing
			}
			b.WriteString(line + "\n")
		}
		f.Close()
	}
	b.WriteString(fmt.Sprintf("%s=%s\n", name, pipe))
	_ = os.WriteFile(path, []byte(b.String()), 0o644)
}

// regDel removes a name from the demo registry.
func regDel(name string) {
	path := registryPath()
	f, err := os.Open(path)
	if err != nil {
		return
	}
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, name+"=") {
			continue
		}
		lines = append(lines, line)
	}
	f.Close()
	_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func cmdRun(args []string) {
	var (
		startDetached bool
		pipePath      string
		name          string
		shell         string
	)
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
	if name == "" {
		name = fmt.Sprintf("term-%d", os.Getpid())
	}
	if pipePath == "" {
		pipePath = fmt.Sprintf("%s/termctl-%s.pipe", workDir(), name)
	}

	opts := []termctl.Opt{
		termctl.WithName(name),
		termctl.WithControlPipe(pipePath),
	}
	if shell != "" {
		opts = append(opts, termctl.WithShell(shell))
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
	// The caller (this demo, standing in for starfleetctl) records the
	// name->pipe mapping itself.
	regPut(name, pipePath)
	log.Printf("run: name=%s pipe=%s", name, pipePath)

	go signalLoop(h)

	if !startDetached {
		if err := h.Attach(""); err != nil {
			log.Fatalf("initial attach: %v", err)
		}
		log.Printf("run: initial attach done")
	}

	if err := h.Run(); err != nil {
		log.Printf("run: %v", err)
	}
	regDel(name)
}

func cmdCtl(args []string) {
	if len(args) < 2 {
		usage()
		os.Exit(2)
	}
	name := args[0]
	action := args[1]
	display := ""
	if len(args) > 2 {
		display = args[2]
	}
	pipe, ok := regGet(name)
	if !ok {
		log.Fatalf("ctl: no terminal named %q in demo registry", name)
	}
	rem, err := termctl.OpenPipe(pipe)
	if err != nil {
		log.Fatalf("ctl: OpenPipe(%s): %v", pipe, err)
	}
	switch action {
	case "attach":
		if err := rem.Attach(display); err != nil {
			log.Fatalf("ctl: attach: %v", err)
		}
		fmt.Printf("attached %s\n", name)
	case "detach":
		if err := rem.Detach(); err != nil {
			log.Fatalf("ctl: detach: %v", err)
		}
		fmt.Printf("detached %s\n", name)
	case "stop":
		if err := rem.Stop(); err != nil {
			log.Fatalf("ctl: stop: %v", err)
		}
		fmt.Printf("stopped %s\n", name)
	default:
		log.Fatalf("ctl: unknown action %q", action)
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
			// Close tears down the window, terminates the shell and removes
			// the control channel. Run() (in main) observes the shell exit
			// via OnExit and returns, ending the process cleanly.
			_ = h.Close()
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
