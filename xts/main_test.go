package xts

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestMain runs the live (round-trip) xts tests against a throwaway Xvfb server
// so they never touch the developer's real session and are safe in CI. If Xvfb
// is not available it falls back to the existing $DISPLAY.
func TestMain(m *testing.M) {
	xvfb, restore, err := startXvfb()
	if err != nil {
		fmt.Fprintf(os.Stderr, "xts: Xvfb unavailable (%v); using DISPLAY=%q\n", err, os.Getenv("DISPLAY"))
		os.Exit(m.Run())
	}
	code := m.Run()
	restore()
	_ = xvfb.Process.Kill()
	_, _ = xvfb.Process.Wait()
	os.Exit(code)
}

// startXvfb launches Xvfb on a server-assigned display, points DISPLAY at it,
// and returns the process plus a func that restores the previous DISPLAY.
func startXvfb() (*exec.Cmd, func(), error) {
	path, err := exec.LookPath("Xvfb")
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.Command(path, "-displayfd", "1", "-screen", "0", "1280x1024x24")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	// Xvfb writes the chosen display number to the -displayfd once it is ready.
	type res struct {
		num string
		err error
	}
	ch := make(chan res, 1)
	go func() {
		line, err := bufio.NewReader(stdout).ReadString('\n')
		ch <- res{strings.TrimSpace(line), err}
	}()

	select {
	case r := <-ch:
		if r.err != nil || r.num == "" {
			_ = cmd.Process.Kill()
			return nil, nil, fmt.Errorf("reading Xvfb display: %v", r.err)
		}
		prev, had := os.LookupEnv("DISPLAY")
		os.Setenv("DISPLAY", ":"+r.num)
		restore := func() {
			if had {
				os.Setenv("DISPLAY", prev)
			} else {
				os.Unsetenv("DISPLAY")
			}
		}
		return cmd, restore, nil
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		return nil, nil, fmt.Errorf("Xvfb did not start within 10s")
	}
}
