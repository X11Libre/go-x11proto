package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// TestBigRequest sends a ChangeProperty far larger than the 65535-unit limit of
// the 16-bit request length, which only works via the BIG-REQUESTS extension
// (auto-negotiated on connect). It reads the property back to confirm the whole
// request was delivered, and checks the connection stays healthy.
func TestBigRequest(t *testing.T) {
	c := connect(t)
	defer c.Close()

	atom, err := rpc.InternAtom(c, "_GOX11_BIGREQ")
	must(t, err, "InternAtom")
	win, err := rpc.CreateWindow1(c, c.DefaultRoot(), -10, -10, 1, 1, 0)
	must(t, err, "CreateWindow1")

	// 100000 CARD32 = ~100002 units, well past 0xffff (needs BIG-REQUESTS)
	const n = 100000
	data := make([]base.CARD32, n)
	for i := range data {
		data[i] = base.CARD32(i)
	}
	if err := rpc.ChangeProperty32(c, 0, win, atom, 6 /*CARDINAL*/, data); err != nil {
		t.Fatalf("big ChangeProperty failed (BIG-REQUESTS not working?): %v", err)
	}

	rep, err := rpc.GetProperty(c, false, win, atom, 0, 0, n)
	must(t, err, "GetProperty")
	if len(rep.Value) != n*4 {
		t.Fatalf("read back %d bytes, want %d", len(rep.Value), n*4)
	}
	if rep.BytesAfter != 0 {
		t.Errorf("BytesAfter = %d, want 0 (incomplete read)", rep.BytesAfter)
	}
	// spot-check the first and last value (stored in the connection's byte order)
	first := decodeCARD32(rep.Value[0:4], c.BE)
	last := decodeCARD32(rep.Value[(n-1)*4:n*4], c.BE)
	if first != 0 || last != n-1 {
		t.Errorf("values: first=%d last=%d, want 0 and %d", first, last, n-1)
	}

	// the connection must still be usable
	if _, err := rpc.InternAtom(c, "_GOX11_BIGREQ_ALIVE"); err != nil {
		t.Errorf("connection unusable after big request: %v", err)
	}
}

func decodeCARD32(b []byte, be bool) uint32 {
	if be {
		return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
	}
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
