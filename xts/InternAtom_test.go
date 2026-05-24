package xts

import (
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"testing"
)

func TestInternAtomLE(t *testing.T) {
	conn := connectLE(t)
	defer conn.Close()

	if atom, err := rpc.InternAtom(conn, "XLIBRE_GO_X11"); err != nil {
		t.Errorf("InternAtom: failed: %s", err)
	} else {
		t.Logf("Created atom: %d\n", atom)
	}
}

func TestInternAtomBE(t *testing.T) {
	conn := connectBE(t)
	defer conn.Close()

	if atom, err := rpc.InternAtom(conn, "XLIBRE_GO_X11"); err != nil {
		t.Errorf("InternAtom (BE) failed: %s", err)
	} else {
		t.Logf("Created atom (BE): %d\n", atom)
	}
}
