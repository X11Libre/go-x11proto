// Package sidplayer plays a Commodore 64 SID tune in the background by shelling
// out to sidplayfp. It is intentionally asset-agnostic: the caller hands it the
// raw SID bytes, so the embedded tune stays with whoever owns the assets.
package sidplayer

import (
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Player runs a single background sidplayfp process. The zero value is ready to
// use; use New for symmetry with the rest of the demo.
type Player struct {
	proc    *os.Process
	tmpFile string // unique SID file, removed by Stop
}

// New returns an idle player.
func New() *Player {
	return &Player{}
}

// Start writes data to a unique temp file and plays it in the background via
// sidplayfp. It is a no-op (returns nil) if sidplayfp is not installed or the
// player cannot be launched; music is a nicety, not a hard requirement. Any
// previously started tune is stopped (and its temp file removed) first.
func (p *Player) Start(data []byte) error {
	p.Stop()
	sidPath, err := exec.LookPath("sidplayfp")
	if err != nil {
		return nil // sidplayfp not installed: silently skip music
	}
	f, err := os.CreateTemp("", "go-x11proto-tetris-*.sid")
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	cmd := exec.Command(sidPath, "-t0", f.Name())
	cmd.Stdin, _ = os.Open(os.DevNull)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	// Run in its own session so terminal signals (Ctrl-C, etc.) and tty access
	// go to the game, not the player; we own its lifecycle via Stop.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		os.Remove(f.Name())
		return err
	}
	p.proc = cmd.Process
	p.tmpFile = f.Name()
	return nil
}

// Stop kills the background player if it is running and removes its temp file.
func (p *Player) Stop() {
	if p.proc != nil {
		p.proc.Signal(os.Kill)
		p.proc.Wait()
		p.proc = nil
	}
	if p.tmpFile != "" {
		os.Remove(p.tmpFile)
		p.tmpFile = ""
	}
}
