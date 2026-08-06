package termctl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// controller is the common surface for external control of a TermHandle. Two
// implementations exist:
//
//   - fifoCtrl: a named pipe created by New at a caller-chosen path; other
//     processes drive it by writing commands to the pipe path directly, or via
//     OpenPipe(path). The caller owns any name->path bookkeeping.
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
	case "dump":
		lines := h.ScreenDump()
		for _, line := range lines {
			h.ctrl.reply("%s", line)
		}
	case "dump-json":
		lines := h.ScreenDump()
		var sb strings.Builder
		sb.WriteString("[")
		for i, line := range lines {
			if i > 0 {
				sb.WriteString(",")
			}
			// Escape the line for JSON
			escaped := strings.ReplaceAll(line, `\`, `\\`)
			escaped = strings.ReplaceAll(escaped, `"`, `\"`)
			sb.WriteString(`"`)
			sb.WriteString(escaped)
			sb.WriteString(`"`)
		}
		sb.WriteString("]")
		h.ctrl.reply("%s", sb.String())
	case "dump-to":
		// dump-to <path> writes screen content to a file as JSON
		if arg == "" {
			h.ctrl.reply("error: dump-to requires a file path")
		} else {
			lines := h.ScreenDump()
			data, err := json.Marshal(lines)
			if err != nil {
				h.ctrl.reply("error: marshal: %v", err)
			} else {
				if err := os.WriteFile(arg, data, 0644); err != nil {
					h.ctrl.reply("error: write file: %v", err)
				} else {
					h.ctrl.reply("ok")
				}
			}
		}
	case "dump-scrollback":
		// dump-scrollback <n> <path> writes n scrollback lines to a file as JSON
		fields := strings.Fields(arg)
		if len(fields) < 2 {
			h.ctrl.reply("error: dump-scrollback requires <n> <path>")
		} else {
			n := 100
			if _, err := fmt.Sscanf(fields[0], "%d", &n); err != nil {
				h.ctrl.reply("error: invalid count: %v", err)
			} else {
				lines := h.ScreenDumpScrollback(n)
				data, err := json.Marshal(lines)
				if err != nil {
					h.ctrl.reply("error: marshal: %v", err)
				} else {
					if err := os.WriteFile(fields[1], data, 0644); err != nil {
						h.ctrl.reply("error: write file: %v", err)
					} else {
						h.ctrl.reply("ok")
					}
				}
			}
		}
	case "quit":
		_ = h.Close()
		h.ctrl.reply("bye")
	case "dimensions":
		// dimensions <path> writes dimensions to a file as JSON
		if arg == "" {
			h.ctrl.reply("error: dimensions requires a file path")
		} else {
			rows, cols := h.ScreenDimensions()
			data, err := json.Marshal(map[string]int{"rows": rows, "cols": cols})
			if err != nil {
				h.ctrl.reply("error: marshal: %v", err)
			} else {
				if err := os.WriteFile(arg, data, 0644); err != nil {
					h.ctrl.reply("error: write file: %v", err)
				} else {
					h.ctrl.reply("ok")
				}
			}
		}
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
	// Remove stale pipe if it exists (from a crashed process)
	_ = os.Remove(c.path)
	if err := mkfifo(c.path); err != nil && !os.IsExist(err) {
		return err
	}
	// Open O_RDWR (not O_RDONLY): a read-only open of a FIFO blocks until a
	// writer appears, and a non-blocking read-only open makes bufio.Scanner
	// bail on EAGAIN. O_RDWR never blocks on open and keeps the read end alive
	// so external writers can open O_WRONLY without blocking, while our
	// blocking reads still see EOF only when we close the fd.
	f, err := os.OpenFile(c.path, os.O_RDWR, 0)
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
	// Use a timeout to avoid blocking forever on a stale pipe with no reader.
	done := make(chan error, 1)
	go func() {
		f, err := os.OpenFile(c.path, os.O_WRONLY, 0)
		if err != nil {
			done <- err
			return
		}
		defer f.Close()
		_, err = fmt.Fprintln(f, cmd)
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("termctl: write timeout (no reader on pipe %s)", c.path)
	}
}

// reply is a no-op for FIFO mode: writing a response back into the same FIFO
// would be read again by our own readLoop (echo loop). External controllers
// driving a FIFO send commands only; they do not read responses. (The fdCtrl
// variant, backed by a dedicated parent-owned fd, may reply.)
func (c *fifoCtrl) reply(format string, args ...any) {}

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
