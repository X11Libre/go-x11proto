package termctl

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPipeStopHeadless exercises the control-pipe path WITHOUT an X connection:
// a handle is created with WithControlPipe, a "stop" command is written to the
// pipe, and we verify the shell exits (Run returns). This isolates the FIFO
// read loop and registry from any X attach/detach logic.
func TestPipeStopHeadless(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "ctl")
	name := "headless-" + filepath.Base(dir)

	// Use an interactive shell: term.Term starts Shell as exec.Command(Shell)
	// with no arguments, so non-interactive programs (e.g. /bin/sleep) exit
	// immediately. /bin/sh stays alive waiting on the PTY.
	h, err := New(WithName(name), WithControlPipe(pipe), WithShell("/bin/sh"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- h.Run() }()

	// Give the read loop time to open the FIFO and the registry to be written.
	time.Sleep(200 * time.Millisecond)

	if _, err := os.Stat(pipe); err != nil {
		t.Fatalf("pipe not created: %v", err)
	}

	// Drive via the registry: Open(name) should find the pipe.
	rem, err := Open(name)
	if err != nil {
		t.Fatalf("Open(%q): %v", name, err)
	}
	if err := rem.Stop(); err != nil {
		t.Fatalf("remote Stop: %v", err)
	}

	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after stop command")
	}

	// Registry entry must be gone after Close.
	if _, err := Open(name); err == nil {
		t.Fatal("registry entry still present after Close")
	}
}
