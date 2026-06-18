// Package sidplayer plays a Commodore 64 SID tune in the background by shelling
// out to sidplayfp. It is intentionally asset-agnostic: the caller hands it the
// raw SID bytes, so the embedded tune stays with whoever owns the assets.
package sidplayer

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var proc *os.Process

// Start writes data to a temp file and plays it in the background via
// sidplayfp. It is a no-op (returns nil) if sidplayfp is not installed or the
// player cannot be launched; music is a nicety, not a hard requirement.
func Start(data []byte) error {
	sidPath, err := exec.LookPath("sidplayfp")
	if err != nil {
		return nil // sidplayfp not installed: silently skip music
	}
	tmpFile := filepath.Join(os.TempDir(), "go-x11proto-tetris.sid")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	cmd := exec.Command(sidPath, "-t0", tmpFile)
	cmd.Stdin, _ = os.Open(os.DevNull)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	// Run in its own session so terminal signals (Ctrl-C, etc.) and tty access
	// go to the game, not the player; we own its lifecycle via Stop.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	proc = cmd.Process
	return nil
}

// Stop kills the background player if it is running.
func Stop() {
	if proc != nil {
		proc.Signal(os.Kill)
		proc.Wait()
		proc = nil
	}
}
