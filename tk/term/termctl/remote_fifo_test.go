package termctl

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestSendToPipeWithoutReaderReturnsPromptly is a regression test for a thread
// leak that took the whole web frontend down.
//
// A ship's control pipe is a FIFO. When the owning process is gone, the FIFO
// file still exists but nobody reads it. Opening a FIFO for writing blocks
// until a reader shows up, so send() used to park a goroutine in open(2)
// forever: the caller got its "no reader" error from a two-second select, but
// the goroutine was abandoned while still blocked. Every call permanently
// consumed an OS thread on the calling process, and after a few thousand calls
// the process hit its file-descriptor limit and could no longer accept() at
// all — the symptom seen in production was a web server that was listening but
// hung, logging "accept4: too many open files".
//
// The fix opens the FIFO with O_WRONLY|O_NONBLOCK, which fails immediately
// with ENXIO when there is no reader. This test pins that down: the call must
// return far faster than the old two-second timeout, and it must say why.
func TestSendToPipeWithoutReaderReturnsPromptly(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "orphan.pipe")

	// A FIFO that nobody ever opens for reading — exactly the state a dead
	// ship leaves behind.
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatalf("Mkfifo: %v", err)
	}
	if _, err := os.Stat(pipe); err != nil {
		t.Fatalf("Stat on the fresh FIFO should succeed, it exists: %v", err)
	}

	rem, err := OpenPipe(pipe)
	if err != nil {
		t.Fatalf("OpenPipe on an existing FIFO must succeed, it only stats: %v", err)
	}

	start := time.Now()
	err = rem.send("status")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("send to a pipe without a reader must fail")
	}
	if !strings.Contains(err.Error(), "no reader") {
		t.Errorf("error should name the real cause, got: %v", err)
	}

	// The old implementation needed its full 2s select timeout to give up.
	// Anything close to that means the blocking open is back.
	const limit = 500 * time.Millisecond
	if elapsed > limit {
		t.Errorf("send blocked for %v, want < %v: the non-blocking open is not in effect", elapsed, limit)
	}
}

// TestSendRepeatedlyDoesNotLeakGoroutines hammers the reader-less path and
// checks the caller's own goroutine count stays flat. A leaked goroutine per
// call would show up here as growth proportional to the iteration count.
func TestSendRepeatedlyDoesNotLeakGoroutines(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "orphan2.pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatalf("Mkfifo: %v", err)
	}
	rem, err := OpenPipe(pipe)
	if err != nil {
		t.Fatalf("OpenPipe: %v", err)
	}

	const rounds = 200
	// Warm up so one-off runtime goroutines do not count as growth.
	for i := 0; i < 5; i++ {
		_ = rem.send("status")
	}
	before := runtime.NumGoroutine()

	for i := 0; i < rounds; i++ {
		_ = rem.send("status")
	}
	after := runtime.NumGoroutine()

	// A leak per call would be +200. Allow slack for unrelated runtime noise.
	const allowed = 20
	if grew := after - before; grew > allowed {
		t.Errorf("goroutines grew by %d over %d sends, want <= %d: per-call leak is back",
			grew, rounds, allowed)
	}
}

// TestFifoCtrlWriteToPipeWithoutReaderReturnsPromptly covers the second copy of
// the same defect, in fifoCtrl.write (control.go). It had the identical shape:
// a goroutine blocked in a blocking O_WRONLY open, wrapped in a select whose
// two-second timeout only protected the caller. A comment there even claimed
// the timeout avoided "blocking forever on a stale pipe with no reader", which
// is exactly what it did not do.
//
// This path is what a long-lived termctl server uses to answer its control
// pipe, so a stale pipe leaks a thread there too.
func TestFifoCtrlWriteToPipeWithoutReaderReturnsPromptly(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "ctrl.pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Fatalf("Mkfifo: %v", err)
	}

	c := &fifoCtrl{path: pipe, done: make(chan struct{})}

	start := time.Now()
	err := c.write("status")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("write to a pipe without a reader must fail")
	}
	if !strings.Contains(err.Error(), "no reader") {
		t.Errorf("error should name the real cause, got: %v", err)
	}
	const limit = 500 * time.Millisecond
	if elapsed > limit {
		t.Errorf("write blocked for %v, want < %v: the non-blocking open is not in effect", elapsed, limit)
	}
}
