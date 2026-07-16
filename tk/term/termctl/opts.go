package termctl

import "os"

// Opt configures a TermHandle at construction time.
type Opt func(*TermHandle)

// WithName assigns a caller-chosen name used for cross-process addressing
// (see Open). Names are the caller's responsibility; collisions are reported
// by New/Open when a registry entry already exists.
func WithName(name string) Opt {
	return func(h *TermHandle) { h.name = name }
}

// WithShell overrides the shell command (defaults to $SHELL, then /bin/sh).
func WithShell(shell string) Opt {
	return func(h *TermHandle) { h.shell = shell }
}

// WithExtraEnv appends environment assignments (e.g. "FOO=bar") to the
// spawned shell.
func WithExtraEnv(env []string) Opt {
	return func(h *TermHandle) { h.extraEnv = env }
}

// WithTitle sets the X window title.
func WithTitle(title string) Opt {
	return func(h *TermHandle) { h.title = title }
}

// WithTTFPath overrides the antialiased font path.
func WithTTFPath(path string) Opt {
	return func(h *TermHandle) { h.ttfPath = path }
}

// WithGeometry sets the desired window size/position for the next Attach.
func WithGeometry(g Geometry) Opt {
	return func(h *TermHandle) { h.geom = g }
}

// WithOnExit registers a cleanup callback invoked once, after the shell has
// exited and before Close returns. Use it to remove pipes, status files, etc.
func WithOnExit(fn func()) Opt {
	return func(h *TermHandle) { h.onExit = fn }
}

// WithControlPipe enables external control via a named pipe at the given
// path. The pipe is created by New and removed by Close. Another process can
// drive this handle through Open(name) (registry) or by writing commands
// directly to the pipe. The name is registered so it can be looked up.
func WithControlPipe(path string) Opt {
	return func(h *TermHandle) { h.ctrl = newFifoCtrl(path, h) }
}

// WithControlFD enables external control over an already-open file descriptor
// (typically inherited from a parent that spawned the terminal, e.g. via the
// TERM_CTRL_FD env convention). The fd is read for commands and written for
// replies; it is NOT closed by Close (the owner is responsible for it). Unlike
// WithControlPipe, no registry entry is created, since the parent already knows
// the fd.
func WithControlFD(fd int) Opt {
	return func(h *TermHandle) { h.ctrl = newFdCtrl(fd, h) }
}

// shell resolves the shell command from the option or the environment.
func (h *TermHandle) shellCmd() string {
	if h.shell != "" {
		return h.shell
	}
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/sh"
}
