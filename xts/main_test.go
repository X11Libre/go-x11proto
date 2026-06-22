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

// TestMain runs the live (round-trip) xts tests against a throwaway X server so
// they never touch the developer's real session and are safe in CI. If the
// server cannot be started it falls back to the existing $DISPLAY.
//
// The launch is configurable via the environment, so the same suite can be
// pointed at a freshly-built X server (e.g. in the xlibre/xserver CI pipeline):
//
//	XTS_XSERVER       server executable (default "Xvfb"); a name resolved via
//	                  $PATH or an absolute/relative path. The special value
//	                  "none" skips spawning and runs against the existing
//	                  $DISPLAY (NB: the live tests are destructive - don't point
//	                  this at a session you care about).
//	XTS_XSERVER_ARGS  arguments other than -displayfd, whitespace-separated
//	                  (default "-screen 0 1280x1024x24")
//
// The harness always adds "-displayfd 1" itself and reads the assigned display
// number from the server's stdout; that mechanism is implemented by every
// server built from the xserver tree (Xvfb, Xephyr, Xnest, Xorg, Xwayland).
func TestMain(m *testing.M) {
	switch strings.ToLower(os.Getenv("XTS_XSERVER")) {
	case "none", "off", "-":
		fmt.Fprintf(os.Stderr, "xts: using existing DISPLAY=%q (no server spawned)\n", os.Getenv("DISPLAY"))
		os.Exit(m.Run())
	}

	srv, restore, err := startXServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "xts: X server unavailable (%v); using DISPLAY=%q\n", err, os.Getenv("DISPLAY"))
		os.Exit(m.Run())
	}
	code := m.Run()
	restore()
	_ = srv.Process.Kill()
	_, _ = srv.Process.Wait()
	os.Exit(code)
}

// startXServer launches the configured X server on a server-assigned display,
// points DISPLAY at it, and returns the process plus a func that restores the
// previous DISPLAY.
func startXServer() (*exec.Cmd, func(), error) {
	exe := os.Getenv("XTS_XSERVER")
	if exe == "" {
		exe = "Xvfb"
	}
	extra := "-screen 0 1280x1024x24"
	if v, ok := os.LookupEnv("XTS_XSERVER_ARGS"); ok {
		extra = v
	}

	path, err := exec.LookPath(exe)
	if err != nil {
		return nil, nil, err
	}
	args := append([]string{"-displayfd", "1"}, strings.Fields(extra)...)
	cmd := exec.Command(path, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	// The server writes the chosen display number to the -displayfd once ready.
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
			return nil, nil, fmt.Errorf("reading %s display: %v", exe, r.err)
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
		return nil, nil, fmt.Errorf("%s did not start within 10s", exe)
	}
}
