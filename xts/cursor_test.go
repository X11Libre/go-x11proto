package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

func TestCursor(t *testing.T) {
	c := connect(t)
	defer c.Close()

	src, err := rpc.CreatePixmap(c, 1, c.DefaultRoot(), 16, 16)
	must(t, err, "CreatePixmap source")
	mask, err := rpc.CreatePixmap(c, 1, c.DefaultRoot(), 16, 16)
	must(t, err, "CreatePixmap mask")

	white := [3]base.CARD16{0xFFFF, 0xFFFF, 0xFFFF}
	black := [3]base.CARD16{0, 0, 0}
	green := [3]base.CARD16{0, 0xFFFF, 0}

	cur, err := rpc.CreateCursor(c, src, mask, white, black, 0, 0)
	must(t, err, "CreateCursor")
	must(t, rpc.RecolorCursor(c, cur, green, black), "RecolorCursor")
	must(t, rpc.FreeCursor(c, cur), "FreeCursor")
	must(t, rpc.FreePixmap(c, src), "FreePixmap src")
	must(t, rpc.FreePixmap(c, mask), "FreePixmap mask")
}

func TestGetMotionEvents(t *testing.T) {
	c := connect(t)
	defer c.Close()
	if _, err := rpc.GetMotionEvents(c, c.DefaultRoot(), 0, 0); err != nil {
		t.Errorf("GetMotionEvents: %v", err)
	}
}
