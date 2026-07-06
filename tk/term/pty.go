//go:build linux

package term

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// PTY is a Linux pseudo-terminal pair. Master is what a Term reads terminal
// output from and writes keyboard input to; Slave becomes the spawned
// shell's controlling terminal until Spawn hands it off and closes the
// parent's copy.
type PTY struct {
	Master, Slave *os.File
}

// OpenPTY allocates a fresh pseudo-terminal pair via /dev/ptmx: unlocks it
// (TIOCSPTLCK) and opens the resulting /dev/pts/N slave. Linux's devpts
// mounts the slave already owned/permissioned for the opening user, so no
// separate grantpt step is needed.
func OpenPTY() (*PTY, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("term: open /dev/ptmx: %w", err)
	}
	fd := int(master.Fd())
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		master.Close()
		return nil, fmt.Errorf("term: unlockpt: %w", err)
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		master.Close()
		return nil, fmt.Errorf("term: ptsname: %w", err)
	}
	slaveName := fmt.Sprintf("/dev/pts/%d", n)
	slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
	if err != nil {
		master.Close()
		return nil, fmt.Errorf("term: open %s: %w", slaveName, err)
	}
	return &PTY{Master: master, Slave: slave}, nil
}

// Resize sets the PTY's window size (delivering SIGWINCH to its foreground
// process group) — call it whenever the Term widget's own pixel size
// changes and the new size maps to a different row/column count.
func (p *PTY) Resize(rows, cols int) error {
	return unix.IoctlSetWinsize(int(p.Master.Fd()), unix.TIOCSWINSZ, &unix.Winsize{
		Row: uint16(rows), Col: uint16(cols),
	})
}

// Close closes the master end (and the slave too, if Spawn hasn't already
// handed it off and closed the parent's copy).
func (p *PTY) Close() error {
	if p.Slave != nil {
		_ = p.Slave.Close()
		p.Slave = nil
	}
	return p.Master.Close()
}

// Spawn starts shell as the PTY's controlling process: stdio is the slave
// end, TERM is set to termType, and it becomes the leader of a new session
// with the slave as its controlling terminal — the standard setup every
// terminal emulator performs for its child shell. After a successful start,
// the parent's copy of the slave fd is closed (the child owns it now); only
// pty.Master is used from here on.
func Spawn(pty *PTY, shell string, extraEnv []string, termType string) (*exec.Cmd, error) {
	cmd := exec.Command(shell)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = pty.Slave, pty.Slave, pty.Slave
	cmd.Env = append(append([]string{}, os.Environ()...), "TERM="+termType)
	cmd.Env = append(cmd.Env, extraEnv...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0, // fd 0 in the child == cmd.Stdin == the slave
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("term: spawn %s: %w", shell, err)
	}
	_ = pty.Slave.Close()
	pty.Slave = nil
	return cmd, nil
}
