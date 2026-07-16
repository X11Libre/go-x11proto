package termctl

import (

	"github.com/X11Libre/go-x11proto/tk/term"
)

// startShell builds the Term (no X resources yet) and spawns the shell in a
// PTY. The window is created later by Attach.
func (h *TermHandle) startShell() error {
	t := &term.Term{
		Type:     term.XTerm256Color,
		Shell:    h.shellCmd(),
		ExtraEnv: h.extraEnv,
		FgRGB:    [3]byte{0xff, 0xff, 0xff},
		BgRGB:    [3]byte{0x00, 0x00, 0x00},
		OnTitle:  func(string) {},
		OnExit:   h.onShellExit,
	}
	if err := t.InitTerm(); err != nil {
		return err
	}
	if err := t.Start(); err != nil {
		return err
	}
	h.t = t
	return nil
}

// onShellExit is invoked from term.Start's wait goroutine once the shell
// process has exited. It records the exit so a later Close knows the shell is
// already gone (Close still tears down the control channel / registry).
func (h *TermHandle) onShellExit(err error) {
	h.mu.Lock()
	h.shellExited = true
	h.mu.Unlock()
}
