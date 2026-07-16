package termctl

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// controller is the common surface for external control of a TermHandle. Two
// implementations exist:
//
//   - fifoCtrl: a named pipe created by New at a caller-chosen path; other
//     processes address it by name via the registry (Open) or by writing to the
//     pipe path directly.
//   - fdCtrl: an already-open file descriptor (typically inherited from a
//     parent that spawned the terminal, e.g. via TERM_CTRL_FD). The parent owns
//     the other end; the child just reads commands and writes replies here.
//
// Commands are newline-terminated text and mirror the Go API:
//
//	attach [display]   show the terminal on display ("" = $DISPLAY)
//	detach            hide the window, keep the shell
//	stop              terminate the shell
//	status            reply "attached" or "detached"
//	quit              detach + stop + close
type controller interface {
	// open starts the read loop. For FIFO it (re)creates the pipe.
	open() error
	// close stops the read loop and releases resources.
	close()
	// write sends a command (used by remote controllers via Open/fd).
	write(cmd string) error
	// reply writes a line back to the controller (best-effort).
	reply(format string, args ...any)
}

// dispatch parses a single command line and acts on the handle. It is shared
// by both controller implementations.
func (h *TermHandle) dispatchCtrl(line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	cmd := fields[0]
	arg := ""
	if len(fields) > 1 {
		arg = strings.Join(fields[1:], " ")
	}
	switch cmd {
	case "attach":
		if err := h.Attach(arg); err != nil {
			h.ctrl.reply("error: %v", err)
		} else {
			h.ctrl.reply("attached")
		}
	case "detach":
		if err := h.Detach(); err != nil {
			h.ctrl.reply("error: %v", err)
		} else {
			h.ctrl.reply("detached")
		}
	case "stop":
		_ = h.Stop()
		h.ctrl.reply("stopped")
	case "status":
		if h.IsAttached() {
			h.ctrl.reply("attached")
		} else {
			h.ctrl.reply("detached")
		}
	case "quit":
		_ = h.Close()
		h.ctrl.reply("bye")
	default:
		h.ctrl.reply("error: unknown command %q", cmd)
	}
}

// fifoCtrl is a named-pipe controller.
type fifoCtrl struct {
	path string
	h    *TermHandle
	file *os.File // reader end
	done chan struct{}
}

func newFifoCtrl(path string, h *TermHandle) *fifoCtrl {
	return &fifoCtrl{path: path, h: h, done: make(chan struct{})}
}

func (c *fifoCtrl) open() error {
	if err := ensureDir(dirOf(c.path)); err != nil {
		return err
	}
	if err := mkfifo(c.path); err != nil && !os.IsExist(err) {
		return err
	}
	f, err := os.OpenFile(c.path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	c.file = f
	go c.readLoop()
	return nil
}

func (c *fifoCtrl) readLoop() {
	sc := bufio.NewScanner(c.file)
	for sc.Scan() {
		c.h.dispatchCtrl(strings.TrimSpace(sc.Text()))
	}
	close(c.done)
}

func (c *fifoCtrl) write(cmd string) error {
	f, err := os.OpenFile(c.path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, cmd)
	return err
}

func (c *fifoCtrl) reply(format string, args ...any) {
	f, err := os.OpenFile(c.path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", args...)
}

func (c *fifoCtrl) close() {
	if c.file != nil {
		_ = c.file.Close()
	}
	<-c.done
	_ = os.Remove(c.path)
}

// fdCtrl is a controller backed by an already-open file descriptor (read cmd,
// write replies). The fd is NOT closed on close() — the caller owns it.
type fdCtrl struct {
	fd   int
	h    *TermHandle
	file *os.File
	done chan struct{}
}

func newFdCtrl(fd int, h *TermHandle) *fdCtrl {
	return &fdCtrl{fd: fd, h: h, done: make(chan struct{})}
}

func (c *fdCtrl) open() error {
	f := os.NewFile(uintptr(c.fd), "ctrl")
	if f == nil {
		return fmt.Errorf("invalid control fd %d", c.fd)
	}
	c.file = f
	go c.readLoop()
	return nil
}

func (c *fdCtrl) readLoop() {
	sc := bufio.NewScanner(c.file)
	for sc.Scan() {
		c.h.dispatchCtrl(strings.TrimSpace(sc.Text()))
	}
	close(c.done)
}

func (c *fdCtrl) write(cmd string) error {
	f := os.NewFile(uintptr(c.fd), "ctrl")
	if f == nil {
		return fmt.Errorf("invalid control fd %d", c.fd)
	}
	defer f.Close()
	_, err := fmt.Fprintln(f, cmd)
	return err
}

func (c *fdCtrl) reply(format string, args ...any) {
	f := os.NewFile(uintptr(c.fd), "ctrl")
	if f == nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", args...)
}

func (c *fdCtrl) close() {
	if c.file != nil {
		_ = c.file.Close()
	}
	<-c.done
}
