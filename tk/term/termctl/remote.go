package termctl

import (
	"fmt"
	"os"
)

// Remote is a handle to a TermHandle owned by another process, driven over its
// control pipe. The caller (e.g. starfleetctl) is responsible for knowing the
// pipe path and for any name->path bookkeeping; termctl only speaks the wire
// protocol once handed a path.
//
// Commands are newline-terminated text sent to the pipe:
//
//	attach [display]   show the terminal on display ("" = $DISPLAY)
//	detach            hide the window, keep the shell
//	stop              terminate the shell
//	status            reply "attached" or "detached" (see below)
//
// Note: FIFO-mode servers do not echo responses back into the same pipe (that
// would loop), so Status cannot read a reply over the pipe. Use IsAttached on
// the owning handle instead; Status here returns an error indicating the reply
// is unavailable.
type Remote struct {
	pipe string
}

// OpenPipe returns a Remote controller for a TermHandle owned by another
// process, addressed by its control-pipe path. The caller supplies the path
// (and, if it needs name-based lookup, maintains the name->path mapping
// itself). It does not own the terminal; commands are forwarded over the pipe.
func OpenPipe(pipe string) (*Remote, error) {
	if pipe == "" {
		return nil, fmt.Errorf("termctl: OpenPipe: empty pipe path")
	}
	// Best-effort liveness check: the pipe must at least exist.
	if _, err := os.Stat(pipe); err != nil {
		return nil, fmt.Errorf("termctl: OpenPipe: %w", err)
	}
	return &Remote{pipe: pipe}, nil
}

// Attach shows the remote terminal on display.
func (r *Remote) Attach(display string) error {
	return r.send("attach " + display)
}

// Detach hides the remote terminal's window.
func (r *Remote) Detach() error {
	return r.send("detach")
}

// Stop terminates the remote terminal's shell.
func (r *Remote) Stop() error {
	return r.send("stop")
}

// Status is not supported over a FIFO control pipe (the server does not echo
// responses back into the same pipe). It always returns an error; query the
// owning handle's IsAttached instead.
func (r *Remote) Status() (string, error) {
	return "", fmt.Errorf("termctl: Remote.Status not supported over pipe; query the owner")
}

// Pipe returns the control-pipe path this Remote drives.
func (r *Remote) Pipe() string { return r.pipe }

func (r *Remote) send(cmd string) error {
	f, err := os.OpenFile(r.pipe, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, cmd)
	return err
}
