package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	"github.com/X11Libre/go-x11proto/tk/dialog"
	"github.com/X11Libre/go-x11proto/tk/font"
)

// TestTkWMDelete checks that a floating dialog advertises WM_DELETE_WINDOW and
// treats the window-manager close message as a cancel (instead of letting the
// WM kill the connection).
func TestTkWMDelete(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	f, err := font.Open(c, "fixed")
	must(t, err, "font.Open")
	defer f.Close(c)

	closed := ""
	fp := &dialog.FilePicker{
		Window:   tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 300, H: 200},
		Font:     f,
		Floating: true,
		OnCancel: func() { closed = "cancel" },
	}
	must(t, fp.Init(), "FilePicker.Init")
	must(t, fp.Open(t.TempDir()), "Open")

	// the WM_PROTOCOLS property must list WM_DELETE_WINDOW
	proto, err := rpc.InternAtom(c, "WM_PROTOCOLS")
	must(t, err, "intern WM_PROTOCOLS")
	del, err := rpc.InternAtom(c, "WM_DELETE_WINDOW")
	must(t, err, "intern WM_DELETE_WINDOW")
	rep, err := rpc.GetProperty(c, false, fp.XID, proto, 0, 0, 16)
	must(t, err, "GetProperty WM_PROTOCOLS")
	if len(rep.Value) < 4 {
		t.Fatalf("WM_PROTOCOLS not set (len %d)", len(rep.Value))
	}

	// deliver a synthetic WM_DELETE_WINDOW client message
	cm := &events.ClientMessageEvent{Format: 32, MessageType: proto}
	cm.Window = fp.XID
	cm.Data[0] = base.CARD32(del)
	fp.HandleWindowEvent(cm)

	if closed != "cancel" {
		t.Errorf("WM close should fire OnCancel, got %q", closed)
	}
	must(t, fp.Destroy(), "Destroy")
}
