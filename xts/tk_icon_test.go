package xts

import (
	"image"
	"image/color"
	"testing"

	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
)

// TestTkSetIcon checks SetIcon writes a well-formed _NET_WM_ICON property
// (width, height, then ARGB pixels).
func TestTkSetIcon(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	win := &tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 50, H: 50}
	must(t, win.Create(), "Create")

	// a 2x2 icon: one opaque red pixel, rest transparent
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0xff, A: 0xff}) // red, opaque
	must(t, win.SetIcon(img), "SetIcon")

	atom, err := rpc.InternAtom(c, "_NET_WM_ICON")
	must(t, err, "intern _NET_WM_ICON")
	rep, err := rpc.GetProperty(c, false, win.XID, atom, 0, 0, 64)
	must(t, err, "GetProperty _NET_WM_ICON")

	// expect 2 + 2*2 = 6 CARD32 values = 24 bytes
	if len(rep.Value) != 24 {
		t.Fatalf("property length = %d bytes, want 24", len(rep.Value))
	}
	u32 := func(i int) uint32 {
		b := rep.Value[i*4 : i*4+4]
		if c.BE {
			return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
		}
		return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	}
	if u32(0) != 2 || u32(1) != 2 {
		t.Errorf("size header = %d x %d, want 2x2", u32(0), u32(1))
	}
	if got := u32(2); got != 0xffff0000 { // ARGB: opaque red
		t.Errorf("pixel[0] = %#08x, want 0xffff0000", got)
	}

	must(t, win.Destroy(), "Destroy")
}
