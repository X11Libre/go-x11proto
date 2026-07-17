//go:build linux

package term

import (
	"bufio"
	"strings"
	"testing"
	"time"
)

func TestPTYSpawnAndReadOutput(t *testing.T) {
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("OpenPTY: %v", err)
	}
	cmd, err := Spawn(pty, "/bin/echo", nil, []string{"PTY_TEST=1"}, "dumb")
	if err != nil {
		pty.Close()
		t.Fatalf("Spawn: %v", err)
	}
	defer pty.Close()

	pty.Master.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, err := bufio.NewReader(pty.Master).ReadString('\n')
	if err != nil {
		t.Fatalf("read from PTY master: %v", err)
	}
	if strings.TrimSpace(line) != "" {
		t.Errorf("`echo` with no args should print just a newline, got %q", line)
	}
	if err := cmd.Wait(); err != nil {
		t.Errorf("echo exited with error: %v", err)
	}
}

func TestPTYResizeDoesNotError(t *testing.T) {
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("OpenPTY: %v", err)
	}
	defer pty.Close()
	if err := pty.Resize(40, 100); err != nil {
		t.Errorf("Resize: %v", err)
	}
}

func TestPTYCatEchoesInput(t *testing.T) {
	pty, err := OpenPTY()
	if err != nil {
		t.Fatalf("OpenPTY: %v", err)
	}
	cmd, err := Spawn(pty, "/bin/cat", nil, nil, "dumb")
	if err != nil {
		pty.Close()
		t.Fatalf("Spawn: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		pty.Close()
	}()

	if _, err := pty.Master.Write([]byte("hi\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	pty.Master.SetReadDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(pty.Master)
	// A PTY in cooked mode echoes input back before the program even sees it,
	// so we expect to read "hi\r\n" twice: once as the terminal's own local
	// echo, once as cat's output written back.
	echoed, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read echoed input: %v", err)
	}
	if strings.TrimSpace(echoed) != "hi" {
		t.Fatalf("echoed = %q, want %q", echoed, "hi")
	}
	fromCat, err := r.ReadString('\n')
	if err != nil {
		t.Fatalf("read cat output: %v", err)
	}
	if strings.TrimSpace(fromCat) != "hi" {
		t.Fatalf("cat output = %q, want %q", fromCat, "hi")
	}
}
