package xts

import (
	"testing"

	"github.com/X11Libre/go-x11proto/proto/base"
	"github.com/X11Libre/go-x11proto/proto/core"
	"github.com/X11Libre/go-x11proto/proto/core/request"
	"github.com/X11Libre/go-x11proto/proto/rpc"
)

// createWin creates an InputOutput child of root with the given value mask.
func createWin(t *testing.T, c *core.X11Conn, mask base.CARD32, r *request.CreateWindowRequest) base.WINDOW {
	t.Helper()
	r.Wid = base.WINDOW(c.NextResourceID())
	r.Parent = c.DefaultRoot()
	r.Class = request.WindowClass_InputOutput
	r.ValueMask = mask
	if _, err := c.Send(r); err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}
	return r.Wid
}

func TestWindowGeometryRoundTrip(t *testing.T) {
	c := connect(t)
	defer c.Close()

	w := createWin(t, c, request.CW_BACKGROUND_PIXEL|request.CW_EVENT_MASK,
		&request.CreateWindowRequest{X: 10, Y: 20, Width: 100, Height: 50,
			BackPixel: c.DefaultBlackPixel(), EventMask: 0xFFFF})

	g, err := rpc.GetGeometry(c, w)
	must(t, err, "GetGeometry")
	if g.X != 10 || g.Y != 20 || g.Width != 100 || g.Height != 50 {
		t.Errorf("geometry = (%d,%d %dx%d), want (10,20 100x50)", g.X, g.Y, g.Width, g.Height)
	}
	if g.Root != c.DefaultRoot() {
		t.Errorf("root = %d, want %d", g.Root, c.DefaultRoot())
	}

	a, err := rpc.GetWindowAttributes(c, w)
	must(t, err, "GetWindowAttributes")
	if a.Class != 1 { // InputOutput
		t.Errorf("class = %d, want 1", a.Class)
	}

	tr, err := rpc.QueryTree(c, w)
	must(t, err, "QueryTree")
	if tr.Parent != c.DefaultRoot() {
		t.Errorf("parent = %d, want root %d", tr.Parent, c.DefaultRoot())
	}

	must(t, rpc.DestroyWindow(c, w), "DestroyWindow")
}

func TestConfigureWindow(t *testing.T) {
	c := connect(t)
	defer c.Close()
	w := createWin(t, c, request.CW_BACKGROUND_PIXEL,
		&request.CreateWindowRequest{Width: 100, Height: 50, BackPixel: c.DefaultBlackPixel()})

	must(t, rpc.MapWindow(c, w), "MapWindow")
	must(t, rpc.ConfigureWindow(c, &request.ConfigureWindowRequest{
		Window: w, ValueMask: request.CONFIG_WINDOW_X | request.CONFIG_WINDOW_Y |
			request.CONFIG_WINDOW_WIDTH | request.CONFIG_WINDOW_HEIGHT,
		X: 5, Y: 6, Width: 200, Height: 80,
	}), "ConfigureWindow")

	g, err := rpc.GetGeometry(c, w)
	must(t, err, "GetGeometry")
	if g.X != 5 || g.Y != 6 || g.Width != 200 || g.Height != 80 {
		t.Errorf("after configure = (%d,%d %dx%d), want (5,6 200x80)", g.X, g.Y, g.Width, g.Height)
	}
	must(t, rpc.UnmapWindow(c, w), "UnmapWindow")
	must(t, rpc.DestroyWindow(c, w), "DestroyWindow")
}

func TestChangeWindowAttributes(t *testing.T) {
	c := connect(t)
	defer c.Close()
	w := createWin(t, c, request.CW_EVENT_MASK,
		&request.CreateWindowRequest{Width: 50, Height: 50, EventMask: 0})

	must(t, rpc.ChangeWindowAttributes(c, &request.ChangeWindowAttributesRequest{
		Window: w, ValueMask: request.CW_BACKGROUND_PIXEL, BackPixel: c.DefaultWhitePixel(),
	}), "ChangeWindowAttributes")
	must(t, rpc.DestroyWindow(c, w), "DestroyWindow")
}

func TestReparentAndSubwindows(t *testing.T) {
	c := connect(t)
	defer c.Close()
	parent := createWin(t, c, request.CW_EVENT_MASK,
		&request.CreateWindowRequest{Width: 200, Height: 200, EventMask: 0})
	child := createWin(t, c, request.CW_EVENT_MASK,
		&request.CreateWindowRequest{Width: 50, Height: 50, EventMask: 0})

	must(t, rpc.MapWindow(c, parent), "MapWindow parent")
	must(t, rpc.ReparentWindow(c, child, parent, 5, 5), "ReparentWindow")

	tr, err := rpc.QueryTree(c, child)
	must(t, err, "QueryTree")
	if tr.Parent != parent {
		t.Errorf("after reparent, parent = %d, want %d", tr.Parent, parent)
	}

	must(t, rpc.MapSubwindows(c, parent), "MapSubwindows")
	must(t, rpc.UnmapSubwindows(c, parent), "UnmapSubwindows")
	must(t, rpc.CirculateWindow(c, request.CirculateRaiseLowest, parent), "CirculateWindow")
	must(t, rpc.ChangeSaveSet(c, request.SaveSetInsert, child), "ChangeSaveSet(insert)")
	must(t, rpc.ChangeSaveSet(c, request.SaveSetDelete, child), "ChangeSaveSet(delete)")
	must(t, rpc.DestroyWindow(c, parent), "DestroyWindow")
}
