package xts

import (
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"testing"
)

// TestInternAtom runs in whatever byte order the current pass uses, so the
// harness exercises it in both little- and big-endian mode against a spawned
// server (see TestMain).
func TestInternAtom(t *testing.T) {
	conn := connect(t)
	defer conn.Close()

	if atom, err := rpc.InternAtom(conn, "XLIBRE_GO_X11"); err != nil {
		t.Errorf("InternAtom failed: %s", err)
	} else {
		t.Logf("created atom: %d", atom)
	}
}
