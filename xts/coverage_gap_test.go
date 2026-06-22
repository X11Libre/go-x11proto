package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core/events"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// TestMiscRequests covers XTS-listed requests that the rest of the suite did
// not yet exercise (happy path) and that are safe against the throwaway server.
func TestMiscRequests(t *testing.T) {
	c := connect(t)
	defer c.Close()
	root := c.DefaultRoot()

	// DestroySubwindows: a parent with a real child window underneath it
	parent := createWin(t, c, request.CW_EVENT_MASK, &request.CreateWindowRequest{Width: 80, Height: 80})
	if _, err := rpc.CreateWindow1(c, parent, 0, 0, 20, 20, 0); err != nil {
		t.Fatalf("CreateWindow1 child: %v", err)
	}
	must(t, rpc.DestroySubwindows(c, parent), "DestroySubwindows") // destroys the child
	must(t, rpc.DestroyWindow(c, parent), "DestroyWindow")

	// ConvertSelection (PRIMARY=1, STRING=31); no owner -> SelectionNotify(None)
	req := createWin(t, c, request.CW_EVENT_MASK, &request.CreateWindowRequest{Width: 10, Height: 10})
	must(t, rpc.ConvertSelection(c, req, 1, 31, 0, 0), "ConvertSelection")
	must(t, rpc.DestroyWindow(c, req), "DestroyWindow req")

	// CopyColormapAndFree
	src, err := rpc.CreateColormap(c, request.ColormapAllocNone, root, screen(c).RootVisual)
	must(t, err, "CreateColormap")
	dst, err := rpc.CopyColormapAndFree(c, src) // frees src
	must(t, err, "CopyColormapAndFree")
	must(t, rpc.FreeColormap(c, dst), "FreeColormap")

	// GetInputFocus (no rpc wrapper; sent directly)
	if _, err := c.SendAndWait(&request.GetInputFocusRequest{}); err != nil {
		t.Errorf("GetInputFocus: %v", err)
	}

	// SendEvent: a synthetic ClientMessage to the root
	ev := events.ClientMessageEvent{Window: root, Format: 32}
	if _, err := c.Send(&request.SendEventRequest{
		Destination: root, EventMask: 0, Event: ev.Encode(c.BE),
	}); err != nil {
		t.Errorf("SendEvent: %v", err)
	}

	// ChangeActivePointerGrab (ignored without an active grab, but must not error)
	must(t, rpc.ChangeActivePointerGrab(c, 0, 0, 0), "ChangeActivePointerGrab")
}

// TestServerControlRequests covers the global-state setters (safe on the
// throwaway server). XTS exercises these; we set benign values.
func TestServerControlRequests(t *testing.T) {
	c := connect(t)
	defer c.Close()

	must(t, rpc.SetScreenSaver(c, 0, 0, 2, 2), "SetScreenSaver") // default blanking/exposures
	must(t, rpc.ForceScreenSaver(c, request.ScreenSaverReset), "ForceScreenSaver")
	must(t, rpc.ChangePointerControl(c, 2, 1, 4, true, true), "ChangePointerControl")
	must(t, rpc.SetAccessControl(c, request.AccessControlEnable), "SetAccessControl(enable)")
	must(t, rpc.SetAccessControl(c, request.AccessControlDisable), "SetAccessControl(disable)") // restore

	addr := []byte{192, 168, 0, 1}
	must(t, rpc.ChangeHosts(c, request.HostInsert, request.FamilyInternet, addr), "ChangeHosts(insert)")
	must(t, rpc.ChangeHosts(c, request.HostDelete, request.FamilyInternet, addr), "ChangeHosts(delete)")

	must(t, rpc.SetCloseDownMode(c, request.CloseDownRetainTemporary), "SetCloseDownMode(retain)")
	must(t, rpc.SetCloseDownMode(c, request.CloseDownDestroy), "SetCloseDownMode(destroy)")
}

// TestTextAndGlyphCursor covers font-dependent requests, skipped if the needed
// fonts are unavailable on the server.
func TestTextAndGlyphCursor(t *testing.T) {
	c := connect(t)
	defer c.Close()

	if font, err := rpc.OpenFont(c, "fixed"); err == nil {
		gc, err := rpc.CreateGC1(c, c.DefaultBlackPixel(), c.DefaultWhitePixel(), font)
		must(t, err, "CreateGC1(font)")
		pm := newPixmap(t, c, 64, 16)
		must(t, rpc.ImageText8(c, pm, gc, 2, 12, "hello"), "ImageText8")
		must(t, rpc.FreePixmap(c, pm), "FreePixmap")
		must(t, rpc.FreeGC(c, gc), "FreeGC")
		must(t, rpc.CloseFont(c, font), "CloseFont")
	} else {
		t.Log(`font "fixed" unavailable; skipping ImageText8`)
	}

	if cur, err := rpc.OpenFont(c, "cursor"); err == nil {
		glyph, err := rpc.CreateGlyphCursor(c, cur, cur, 0, 1,
			[3]base.CARD16{0, 0, 0}, [3]base.CARD16{0xffff, 0xffff, 0xffff})
		must(t, err, "CreateGlyphCursor")
		must(t, rpc.FreeCursor(c, glyph), "FreeCursor")
		must(t, rpc.CloseFont(c, cur), "CloseFont(cursor)")
	} else {
		t.Log(`font "cursor" unavailable; skipping CreateGlyphCursor`)
	}
}
