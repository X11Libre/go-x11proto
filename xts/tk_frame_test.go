package xts

import (
	"testing"

	tk_core "github.com/X11Libre/go-x11proto/tk/core"
	tk_widget "github.com/X11Libre/go-x11proto/tk/widget"
)

// TestTkFrame builds a Frame with a top bar, a right bar and a center child,
// lays them out against the live server, and verifies each child's geometry via
// GetGeometry. Layout math is unit-tested offline; this checks the real
// MoveResize path in both byte orders.
func TestTkFrame(t *testing.T) {
	c := connect(t)
	defer c.Close()
	tk := tk_core.MakeTkConn(c)
	tkp := &tk

	frame := &tk_widget.Frame{
		Window: tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, X: 0, Y: 0, W: 200, H: 100},
	}
	// Frame must exist before children can be parented to it.
	must(t, frameCreate(frame), "Frame.Create")

	mkChild := func() *tk_core.Window {
		w := &tk_core.Window{Drawable: tk_core.Drawable{Conn: tkp}, Parent: &frame.Window, X: 0, Y: 0, W: 10, H: 10}
		must(t, w.Create(), "child.Create")
		must(t, w.Map(), "child.Map")
		return w
	}
	topBar := mkChild()
	rightBar := mkChild()
	center := mkChild()

	frame.Top = &tk_widget.Slot{Win: topBar, Extent: 20}
	frame.Right = &tk_widget.Slot{Win: rightBar, Extent: 16}
	frame.Center = center

	frame.Relayout(200, 100)

	checkGeom(t, topBar, 200, 20, "top bar")
	checkGeom(t, rightBar, 16, 80, "right bar")
	checkGeom(t, center, 184, 80, "center")

	// Resize: children must follow.
	frame.Relayout(400, 200)
	checkGeom(t, topBar, 400, 20, "top bar after resize")
	checkGeom(t, center, 384, 180, "center after resize")

	must(t, frame.Destroy(), "Frame.Destroy")
}

// frameCreate creates+maps the frame window without doing the initial layout
// (children don't exist yet at that point in the test).
func frameCreate(f *tk_widget.Frame) error {
	if err := f.Window.Create(); err != nil {
		return err
	}
	return f.Window.Map()
}

func checkGeom(t *testing.T, w *tk_core.Window, wantW, wantH int, what string) {
	t.Helper()
	g, err := w.GetGeometry()
	must(t, err, what+" GetGeometry")
	if int(g.Width) != wantW || int(g.Height) != wantH {
		t.Errorf("%s geometry = %dx%d, want %dx%d", what, g.Width, g.Height, wantW, wantH)
	}
}
