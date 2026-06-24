package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// TestOversizedRequestRejected verifies that a request longer than the server's
// maximum is refused cleanly instead of wrapping its CARD16 length field and
// desyncing the connection (which used to happen with a too-large _NET_WM_ICON).
func TestOversizedRequestRejected(t *testing.T) {
	c := connect(t)
	defer c.Close()

	atom, err := rpc.InternAtom(c, "_GOX11_BIG_TEST")
	must(t, err, "InternAtom")
	win, err := rpc.CreateWindow1(c, c.DefaultRoot(), -10, -10, 1, 1, 0)
	must(t, err, "CreateWindow1")

	// one item per 4-byte unit, comfortably past the server maximum
	huge := make([]base.CARD32, int(c.Setup.MaxRequestSize)+100)
	if err := rpc.ChangeProperty32(c, 0, win, atom, 6 /*CARDINAL*/, huge); err == nil {
		t.Error("oversized ChangeProperty should be rejected, got nil error")
	}

	// the connection must still be usable — a desync would corrupt this too
	if _, err := rpc.InternAtom(c, "_GOX11_STILL_ALIVE"); err != nil {
		t.Errorf("connection unusable after oversized request (desync?): %v", err)
	}
}
