package termctl

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"syscall"
)

// newID returns a short random identifier used when the caller does not set a
// name.
func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "term-" + hex.EncodeToString(b[:])
}

// dirOf returns the directory part of a path.
func dirOf(p string) string {
	return filepath.Dir(p)
}

// mkfifo creates a named pipe at path.
func mkfifo(path string) error {
	return syscall.Mkfifo(path, 0o600)
}

// ensureDir creates a directory if missing.
func ensureDir(p string) error {
	return os.MkdirAll(p, 0o755)
}
