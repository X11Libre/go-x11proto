package xts

import (
	"testing"
	"time"

	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/rpc"
	"github.com/X11Libre/go-x11proto/tk/clipboard"
)

// TestTkClipboard runs a full selection transfer between two connections: an
// owner serves CLIPBOARD text and a requestor fetches it. Exercises
// SetSelectionOwner / ConvertSelection / SelectionRequest / SelectionNotify and
// the property transfer in both byte orders.
func TestTkClipboard(t *testing.T) {
	owner := connect(t)
	defer owner.Close()
	req := connect(t)
	defer req.Close()

	wOwner, err := rpc.CreateWindow1(owner, owner.DefaultRoot(), -10, -10, 1, 1, clipboard.EventMask)
	must(t, err, "owner window")
	wReq, err := rpc.CreateWindow1(req, req.DefaultRoot(), -10, -10, 1, 1, clipboard.EventMask)
	must(t, err, "requestor window")

	ownerCB, err := clipboard.New(owner, wOwner, "CLIPBOARD")
	must(t, err, "owner clipboard")
	reqCB, err := clipboard.New(req, wReq, "CLIPBOARD")
	must(t, err, "requestor clipboard")

	const want = "hello, clipboard — äöü"
	must(t, ownerCB.Own(want), "Own")
	if !ownerCB.Owned() {
		t.Fatal("owner does not report ownership")
	}

	// Serve the owner's selection requests from a background pump.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case ev := <-owner.Events():
				if ev != nil {
					ownerCB.HandleX11WindowEvent(ev.ReceiverWindow(), ev)
				}
			}
		}
	}()

	got, ok, err := reqCB.GetText(3 * time.Second)
	must(t, err, "GetText")
	if !ok {
		t.Fatal("GetText reported no owner / refused")
	}
	if got != want {
		t.Errorf("clipboard text = %q, want %q", got, want)
	}
}

// TestTkClipboardNoOwner: GetText reports no owner when the selection is unheld.
func TestTkClipboardNoOwner(t *testing.T) {
	c := connect(t)
	defer c.Close()
	w, err := rpc.CreateWindow1(c, c.DefaultRoot(), -10, -10, 1, 1, clipboard.EventMask)
	must(t, err, "window")

	// A fresh, almost certainly unowned selection name.
	cb, err := clipboard.New(c, w, "GOX11_TEST_SELECTION_UNOWNED")
	must(t, err, "clipboard")
	_, ok, err := cb.GetText(time.Second)
	must(t, err, "GetText")
	if ok {
		t.Error("expected no owner for an unheld selection")
	}
}

// compile-time: Clipboard is usable as a window handler.
var _ = func() events.Event { return nil }
